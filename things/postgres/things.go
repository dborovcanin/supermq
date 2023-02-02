// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/lib/pq" // required for DB access
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things"
)

const (
	errDuplicate  = "unique_violation"
	errFK         = "foreign_key_violation"
	errInvalid    = "invalid_text_representation"
	errTruncation = "string_data_right_truncation"
	locationKey   = "location"

	meterNum1 = "meter_num_1"
	meterNum2 = "meter_num_2"
)

var _ things.ThingRepository = (*thingRepository)(nil)

type thingRepository struct {
	db Database
}

// NewThingRepository instantiates a PostgreSQL implementation of thing
// repository.
func NewThingRepository(db Database) things.ThingRepository {
	return &thingRepository{
		db: db,
	}
}

func (tr thingRepository) Save(ctx context.Context, ths ...things.Thing) ([]things.Thing, error) {
	tx, err := tr.db.BeginTxx(ctx, nil)
	if err != nil {
		return []things.Thing{}, errors.Wrap(things.ErrCreateEntity, err)
	}

	q := `INSERT INTO things (id, owner, name, key, metadata)
		  VALUES (:id, :owner, :name, :key, :metadata);`

	for _, thing := range ths {
		dbth, err := toDBThing(thing)
		if err != nil {
			return []things.Thing{}, errors.Wrap(things.ErrCreateEntity, err)
		}

		if _, err := tx.NamedExecContext(ctx, q, dbth); err != nil {
			tx.Rollback()
			pqErr, ok := err.(*pq.Error)
			if ok {
				switch pqErr.Code.Name() {
				case errInvalid, errTruncation:
					return []things.Thing{}, errors.Wrap(things.ErrMalformedEntity, err)
				case errDuplicate:
					return []things.Thing{}, errors.Wrap(things.ErrConflict, err)
				}
			}

			return []things.Thing{}, errors.Wrap(things.ErrCreateEntity, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return []things.Thing{}, errors.Wrap(things.ErrCreateEntity, err)
	}

	return ths, nil
}

func (tr thingRepository) Update(ctx context.Context, t things.Thing) error {
	q := `UPDATE things SET name = :name, metadata = :metadata WHERE id = :id;`

	dbth, err := toDBThing(t)
	if err != nil {
		return errors.Wrap(things.ErrUpdateEntity, err)
	}

	res, errdb := tr.db.NamedExecContext(ctx, q, dbth)
	if errdb != nil {
		pqErr, ok := errdb.(*pq.Error)
		if ok {
			switch pqErr.Code.Name() {
			case errInvalid, errTruncation:
				return errors.Wrap(things.ErrMalformedEntity, errdb)
			}
		}

		return errors.Wrap(things.ErrUpdateEntity, errdb)
	}

	cnt, errdb := res.RowsAffected()
	if errdb != nil {
		return errors.Wrap(things.ErrUpdateEntity, errdb)
	}

	if cnt == 0 {
		return things.ErrNotFound
	}

	return nil
}

func (tr thingRepository) UpdateKey(ctx context.Context, owner, id, key string) error {
	q := `UPDATE things SET key = :key WHERE owner = :owner AND id = :id;`

	dbth := dbThing{
		ID:    id,
		Owner: owner,
		Key:   key,
	}

	res, err := tr.db.NamedExecContext(ctx, q, dbth)
	if err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok {
			switch pqErr.Code.Name() {
			case errInvalid:
				return errors.Wrap(things.ErrMalformedEntity, err)
			case errDuplicate:
				return errors.Wrap(things.ErrConflict, err)
			}
		}

		return errors.Wrap(things.ErrUpdateEntity, err)
	}

	cnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(things.ErrUpdateEntity, err)
	}

	if cnt == 0 {
		return things.ErrNotFound
	}

	return nil
}

func (tr thingRepository) RetrieveByID(ctx context.Context, owner, id string) (things.Thing, error) {
	q := `SELECT name, key, metadata FROM things WHERE id = $1;`

	dbth := dbThing{ID: id}

	if err := tr.db.QueryRowxContext(ctx, q, id).StructScan(&dbth); err != nil {
		pqErr, ok := err.(*pq.Error)
		if err == sql.ErrNoRows || ok && errInvalid == pqErr.Code.Name() {
			return things.Thing{}, errors.Wrap(things.ErrNotFound, err)
		}
		return things.Thing{}, errors.Wrap(things.ErrSelectEntity, err)
	}

	return toThing(dbth)
}

func (tr thingRepository) RetrieveByKey(ctx context.Context, key string) (string, error) {
	q := `SELECT id FROM things WHERE key = $1;`

	var id string
	if err := tr.db.QueryRowxContext(ctx, q, key).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return "", errors.Wrap(things.ErrNotFound, err)
		}
		return "", errors.Wrap(things.ErrSelectEntity, err)
	}

	return id, nil
}

func (tr thingRepository) RetrieveByIDs(ctx context.Context, thingIDs []string, pm things.PageMetadata) (things.Page, error) {
	if len(thingIDs) == 0 {
		return things.Page{}, nil
	}

	nq, name := getNameQuery(pm.Name)
	oq := getOrderQuery(pm.Order)
	dq := getDirQuery(pm.Dir)
	idq := fmt.Sprintf("WHERE id IN ('%s') ", strings.Join(thingIDs, "','"))

	m, mq, err := getMetadataQuery(pm.Metadata)
	if err != nil {
		return things.Page{}, errors.Wrap(things.ErrSelectEntity, err)
	}

	q := fmt.Sprintf(`SELECT id, owner, name, key, metadata FROM things
					   %s%s%s ORDER BY %s %s;`, idq, mq, nq, oq, dq)

	params := map[string]interface{}{
		"name":     name,
		"metadata": m,
	}

	rows, err := tr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.Page{}, errors.Wrap(things.ErrSelectEntity, err)
	}
	defer rows.Close()

	var items []things.Thing
	for rows.Next() {
		dbth := dbThing{}
		if err := rows.StructScan(&dbth); err != nil {
			return things.Page{}, errors.Wrap(things.ErrSelectEntity, err)
		}

		th, err := toThing(dbth)
		if err != nil {
			return things.Page{}, errors.Wrap(things.ErrViewEntity, err)
		}

		items = append(items, th)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM things %s%s%s;`, idq, mq, nq)

	total, err := total(ctx, tr.db, cq, params)
	if err != nil {
		return things.Page{}, errors.Wrap(things.ErrSelectEntity, err)
	}

	page := things.Page{
		Things: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Order:  pm.Order,
			Dir:    pm.Dir,
		},
	}

	return page, nil
}

func getOwnerQuery(fetchSharedThings bool) string {
	if fetchSharedThings {
		return ""
	}
	return "owner = :owner"
}

func (tr thingRepository) RetrieveAll(ctx context.Context, owner string, pm things.PageMetadata) (things.Page, error) {
	nq, name := getNameQuery(pm.Name)
	oq := getOrderQuery(pm.Order)
	dq := getDirQuery(pm.Dir)
	ownerQuery := getOwnerQuery(pm.FetchSharedThings)
	m, mq, err := getMetadataQuery(pm.Metadata)
	if err != nil {
		return things.Page{}, errors.Wrap(things.ErrSelectEntity, err)
	}

	var query []string
	if mq != "" {
		query = append(query, mq)
	}
	if nq != "" {
		query = append(query, nq)
	}
	if ownerQuery != "" {
		query = append(query, ownerQuery)
	}

	var ids string
	if len(pm.SharedThings) > 0 {
		ids = fmt.Sprintf("id IN ('%s')", strings.Join(pm.SharedThings, "', '"))
		query = append(query, ids)
	}
	var whereClause string
	if len(query) > 0 {
		whereClause = fmt.Sprintf(" WHERE %s", strings.Join(query, " AND "))
	}

	q := fmt.Sprintf(`SELECT  id, name, key, metadata FROM things
	      %s ORDER BY %s %s LIMIT :limit OFFSET :offset;`, whereClause, oq, dq)
	params := map[string]interface{}{
		"owner":    owner,
		"limit":    pm.Limit,
		"offset":   pm.Offset,
		"name":     name,
		"metadata": m,
	}

	rows, err := tr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.Page{}, errors.Wrap(things.ErrSelectEntity, err)
	}
	defer rows.Close()

	var items []things.Thing
	for rows.Next() {
		dbth := dbThing{Owner: owner}
		if err := rows.StructScan(&dbth); err != nil {
			return things.Page{}, errors.Wrap(things.ErrSelectEntity, err)
		}

		th, err := toThing(dbth)
		if err != nil {
			return things.Page{}, errors.Wrap(things.ErrViewEntity, err)
		}

		items = append(items, th)
	}

	cq := fmt.Sprintf(`SELECT COUNT(*) FROM things %s;`, whereClause)

	total, err := total(ctx, tr.db, cq, params)
	if err != nil {
		return things.Page{}, errors.Wrap(things.ErrSelectEntity, err)
	}

	page := things.Page{
		Things: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Order:  pm.Order,
			Dir:    pm.Dir,
		},
	}

	return page, nil
}

func (tr thingRepository) SearchThingsParams(ctx context.Context, devices []string, modem bool) (things.Page, error) {
	if modem {
		return tr.searchModems(ctx, devices)
	}
	return tr.searchMeters(ctx, devices)
}

func (tr thingRepository) searchModems(ctx context.Context, devices []string) (things.Page, error) {
	q := `SELECT DISTINCT th.id, th.name, th.key, th.metadata, ch.metadata
	FROM things th
	JOIN connections conn on th.id = conn.thing_id
	JOIN channels ch ON conn.channel_id = ch.id
	WHERE th.metadata -> 'watermeter' -> 'sn' <@ '%s'::jsonb`

	var b strings.Builder
	for _, dev := range devices {
		fmt.Fprintf(&b, "\"%s\", ", dev)
	}
	s := b.String()
	s = "[" + s[:b.Len()-2] + "]"
	q = fmt.Sprintf(q, s)

	rows, err := tr.db.QueryContext(ctx, q)
	if err != nil {
		return things.Page{}, errors.Wrap(things.ErrSelectEntity, err)
	}
	defer rows.Close()

	var items []things.Thing
	for rows.Next() {
		dbth := dbThing{}
		meta := []byte{}
		if err := rows.Scan(&dbth.ID, &dbth.Name, &dbth.Key, &dbth.Metadata, &meta); err != nil {
			return things.Page{}, errors.Wrap(things.ErrSelectEntity, err)
		}

		var m metadata
		if err := json.Unmarshal(dbth.Metadata, &m); err != nil {
			return things.Page{}, errors.Wrap(things.ErrMalformedEntity, err)
		}

		th := things.Thing{
			ID:       dbth.ID,
			Owner:    dbth.Owner,
			Name:     dbth.Name,
			Key:      dbth.Key,
			Metadata: m.Value,
		}

		if err != nil {
			return things.Page{}, errors.Wrap(things.ErrViewEntity, err)
		}
		var lm locationMeta
		if len(meta) > 0 {
			if err := json.Unmarshal(meta, &lm); err != nil {
				return things.Page{}, errors.Wrap(things.ErrMalformedEntity, err)
			}
			th.Metadata[locationKey] = lm.Loc.String()
		}
		items = append(items, th)
	}
	ret := []things.Thing{}
	for _, th := range items {
		item := th
		item.ID = th.ID + "#1"
		ret = append(ret, item)
		if _, ok := th.Metadata[meterNum2]; ok {
			item.ID = th.ID + "#2"
			ret = append(ret, item)
		}
	}
	page := things.Page{
		Things: ret,
	}

	return page, nil
}

func (tr thingRepository) searchMeters(ctx context.Context, devices []string) (things.Page, error) {
	q := `SELECT DISTINCT CONCAT(th.id, '#1'), th.name, th.key, th.metadata, ch.metadata
	FROM things th
	JOIN connections conn on th.id = conn.thing_id
	JOIN channels ch ON conn.channel_id = ch.id
	WHERE th.metadata -> 'watermeter' -> '%s' <@ '%s'::jsonb
	UNION ALL
		SELECT DISTINCT CONCAT(th.id, '#2'), th.name, th.key, th.metadata, ch.metadata
		FROM things th
		JOIN connections conn on th.id = conn.thing_id
		JOIN channels ch ON conn.channel_id = ch.id WHERE
		th.metadata -> 'watermeter' -> '%s' <@ '%s'::jsonb;`

	var b strings.Builder
	for _, dev := range devices {
		fmt.Fprintf(&b, "\"%s\", ", dev)
	}
	s := b.String()
	s = "[" + s[:b.Len()-2] + "]"
	q = fmt.Sprintf(q, meterNum1, s, meterNum2, s)

	rows, err := tr.db.QueryContext(ctx, q)
	if err != nil {
		return things.Page{}, errors.Wrap(things.ErrSelectEntity, err)
	}
	defer rows.Close()

	var items []things.Thing
	for rows.Next() {
		dbth := dbThing{}
		meta := []byte{}
		if err := rows.Scan(&dbth.ID, &dbth.Name, &dbth.Key, &dbth.Metadata, &meta); err != nil {
			return things.Page{}, errors.Wrap(things.ErrSelectEntity, err)
		}

		var m metadata
		if err := json.Unmarshal(dbth.Metadata, &m); err != nil {
			return things.Page{}, errors.Wrap(things.ErrMalformedEntity, err)
		}

		th := things.Thing{
			ID:       dbth.ID,
			Owner:    dbth.Owner,
			Name:     dbth.Name,
			Key:      dbth.Key,
			Metadata: m.Value,
		}

		if err != nil {
			return things.Page{}, errors.Wrap(things.ErrViewEntity, err)
		}
		var lm locationMeta
		if len(meta) > 0 {
			if err := json.Unmarshal(meta, &lm); err != nil {
				return things.Page{}, errors.Wrap(things.ErrMalformedEntity, err)
			}
			th.Metadata[locationKey] = lm.Loc.String()
		}
		items = append(items, th)
	}
	page := things.Page{
		Things: items,
	}

	return page, nil
}

func (tr thingRepository) RetrieveByChannel(ctx context.Context, owner, chID string, pm things.PageMetadata) (things.Page, error) {
	oq := getConnOrderQuery(pm.Order, "th")
	dq := getDirQuery(pm.Dir)
	ownerQuery := getOwnerQuery(pm.FetchSharedThings)

	var query []string
	if ownerQuery != "" {
		query = append(query, fmt.Sprintf("th.%s", ownerQuery))
	}
	var ids string
	if len(pm.SharedThings) > 0 {
		ids = fmt.Sprintf("th.id IN ('%s')", strings.Join(pm.SharedThings, "', '"))
		query = append(query, ids)
	}
	var thingWhereClause string
	if len(query) > 0 {
		thingWhereClause = fmt.Sprintf(" WHERE %s", strings.Join(query, " AND "))
	}

	// Verify if UUID format is valid to avoid internal Postgres error
	if _, err := uuid.FromString(chID); err != nil {
		return things.Page{}, errors.Wrap(things.ErrNotFound, err)
	}

	var q, qc string
	switch pm.Disconnected {
	case true:
		if thingWhereClause == "" {
			thingWhereClause = " WHERE "
		}
		q = fmt.Sprintf(`SELECT id, name, key, metadata
		        FROM things th
		        %s AND th.id NOT IN
		        (SELECT id FROM things th
		          INNER JOIN connections conn
		          ON th.id = conn.thing_id
		          WHERE conn.channel_owner = :owner AND conn.channel_id = :channel)
		        ORDER BY %s %s
		        LIMIT :limit
		        OFFSET :offset;`, thingWhereClause, oq, dq)

		qc = fmt.Sprintf(`SELECT COUNT(*)
		        FROM things th
		        %s AND th.id NOT IN
		        (SELECT id FROM things th
		          INNER JOIN connections conn
		          ON th.id = conn.thing_id
		          WHERE conn.channel_owner = $1 AND conn.channel_id = $2);`, thingWhereClause)
		qc = strings.Replace(qc, ":owner", "$1", 1)
	default:
		q = fmt.Sprintf(`SELECT id, name, key, metadata
		        FROM things th
		        INNER JOIN connections conn
		        ON th.id = conn.thing_id
		        WHERE conn.channel_owner = :owner AND conn.channel_id = :channel
		        ORDER BY %s %s
		        LIMIT :limit
		        OFFSET :offset;`, oq, dq)

		qc = `SELECT COUNT(*)
		        FROM things th
		        INNER JOIN connections conn
		        ON th.id = conn.thing_id
		        WHERE conn.channel_owner = $1 AND conn.channel_id = $2 ; `
	}

	params := map[string]interface{}{
		"owner":   owner,
		"channel": chID,
		"limit":   pm.Limit,
		"offset":  pm.Offset,
	}

	rows, err := tr.db.NamedQueryContext(ctx, q, params)
	if err != nil {
		return things.Page{}, errors.Wrap(things.ErrSelectEntity, err)
	}
	defer rows.Close()

	var items []things.Thing
	for rows.Next() {
		dbth := dbThing{Owner: owner}
		if err := rows.StructScan(&dbth); err != nil {
			return things.Page{}, errors.Wrap(things.ErrSelectEntity, err)
		}

		th, err := toThing(dbth)
		if err != nil {
			return things.Page{}, errors.Wrap(things.ErrViewEntity, err)
		}

		items = append(items, th)
	}

	var total uint64
	if err := tr.db.GetContext(ctx, &total, qc, owner, chID); err != nil {
		return things.Page{}, errors.Wrap(things.ErrSelectEntity, err)
	}

	return things.Page{
		Things: items,
		PageMetadata: things.PageMetadata{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}, nil
}

func (tr thingRepository) Remove(ctx context.Context, owner, id string) error {
	dbth := dbThing{
		ID:    id,
		Owner: owner,
	}
	q := `DELETE FROM things WHERE id = :id`
	if _, err := tr.db.NamedExecContext(ctx, q, dbth); err != nil {
		return errors.Wrap(things.ErrRemoveEntity, err)
	}
	return nil
}

type dbThing struct {
	ID       string `db:"id"`
	Owner    string `db:"owner"`
	Name     string `db:"name"`
	Key      string `db:"key"`
	Metadata []byte `db:"metadata"`
}

func toDBThing(th things.Thing) (dbThing, error) {
	data := []byte("{}")
	if len(th.Metadata) > 0 {
		b, err := json.Marshal(th.Metadata)
		if err != nil {
			return dbThing{}, errors.Wrap(things.ErrMalformedEntity, err)
		}
		data = b
	}

	return dbThing{
		ID:       th.ID,
		Owner:    th.Owner,
		Name:     th.Name,
		Key:      th.Key,
		Metadata: data,
	}, nil
}

func toThing(dbth dbThing) (things.Thing, error) {
	var metadata map[string]interface{}
	if err := json.Unmarshal(dbth.Metadata, &metadata); err != nil {
		return things.Thing{}, errors.Wrap(things.ErrMalformedEntity, err)
	}

	return things.Thing{
		ID:       dbth.ID,
		Owner:    dbth.Owner,
		Name:     dbth.Name,
		Key:      dbth.Key,
		Metadata: metadata,
	}, nil
}

type locationMeta struct {
	Loc location `json:"watermeter"`
}

type metadata struct {
	Value map[string]interface{} `json:"watermeter"`
}

type location struct {
	Country string `json:"country"`
	City    string `json:"city"`
	Street  string `json:"street"`
	Number  string `json:"number"`
}

func (l location) String() string {
	return strings.Join([]string{l.Street + " " + l.Number, l.City, l.Country}, ", ")
}
