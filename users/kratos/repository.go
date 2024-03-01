// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package kratos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strconv"

	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/users"
	ory "github.com/ory/client-go"
)

var _ mgclients.Repository = (*repository)(nil)

type repository struct {
	*ory.APIClient
	schemaID string
	hasher   users.Hasher
}

// Repository defines the required dependencies for Client repository.
type Repository interface {
	mgclients.Repository

	// Save persists the client account. A non-nil error is returned to indicate
	// operation failure.
	Save(ctx context.Context, client mgclients.Client) (mgclients.Client, error)

	RetrieveByID(ctx context.Context, id string) (mgclients.Client, error)

	UpdateRole(ctx context.Context, client mgclients.Client) (mgclients.Client, error)

	CheckSuperAdmin(ctx context.Context, adminID string) error
}

func NewRepository(client *ory.APIClient, schemaID string, hasher users.Hasher) Repository {
	return &repository{
		APIClient: client,
		schemaID:  schemaID,
		hasher:    hasher,
	}
}

func (repo *repository) Save(ctx context.Context, user mgclients.Client) (mgclients.Client, error) {
	hashedPassword, err := repo.hasher.Hash(user.Credentials.Secret)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	state := mgclients.ToOryState(user.Status)
	identity, resp, err := repo.IdentityAPI.CreateIdentity(ctx).CreateIdentityBody(
		ory.CreateIdentityBody{
			SchemaId: repo.schemaID,
			Traits: map[string]interface{}{
				"email":      user.Credentials.Identity,
				"username":   user.Name,
				"enterprise": slices.Contains(user.Tags, "enterprise"),
				"newsletter": slices.Contains(user.Tags, "newsletter"),
			},
			State:          &state,
			MetadataPublic: user.Metadata,
			MetadataAdmin: map[string]interface{}{
				"role":        user.Role,
				"permissions": user.Permissions,
			},
			Credentials: &ory.IdentityWithCredentials{
				Password: &ory.IdentityWithCredentialsPassword{
					Config: &ory.IdentityWithCredentialsPasswordConfig{
						HashedPassword: &hashedPassword,
						Password:       &user.Credentials.Secret,
					},
				},
			},
		},
	).Execute()
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrCreateEntity, decodeError(resp))
	}

	return toClient(identity), nil
}

func (repo *repository) RetrieveByID(ctx context.Context, id string) (mgclients.Client, error) {
	identity, resp, err := repo.IdentityAPI.GetIdentity(ctx, id).Execute()
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrViewEntity, decodeError(resp))
	}

	if identity == nil {
		return mgclients.Client{}, repoerr.ErrNotFound
	}

	return toClient(identity), nil
}

func (repo *repository) RetrieveByIdentity(ctx context.Context, identity string) (mgclients.Client, error) {
	identities, resp, err := repo.IdentityAPI.ListIdentities(ctx).PageSize(1).CredentialsIdentifier(identity).Execute()
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrViewEntity, decodeError(resp))
	}

	if len(identities) == 0 || len(identities) != 1 {
		return mgclients.Client{}, repoerr.ErrNotFound
	}

	return toClient(&identities[0]), nil
}

func (repo *repository) RetrieveAll(ctx context.Context, page mgclients.Page) (mgclients.ClientsPage, error) {
	return repo.filterUsers(ctx, page)
}

func (repo *repository) filterUsers(ctx context.Context, page mgclients.Page) (mgclients.ClientsPage, error) {
	identities, resp, err := repo.IdentityAPI.ListIdentities(ctx).Page(0).PerPage(1000).Execute()
	if err != nil {
		return mgclients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, decodeError(resp))
	}
	total, err := strconv.ParseUint(resp.Header.Get("X-Total-Count"), 10, 64)
	if err != nil {
		return mgclients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	clients := []mgclients.Client{}
	for _, identity := range identities {
		client := toClient(&identity)
		if client.Status != mgclients.AllStatus {
			if client.Status != page.Status {
				continue
			}
		}
		if client.Role != mgclients.AllRole {
			if client.Role != page.Role {
				continue
			}
		}
		if page.Name != "" {
			if client.Name != page.Name {
				continue
			}
		}
		if page.Domain != "" {
			if client.Domain != page.Domain {
				continue
			}
		}
		if page.Tag != "" {
			if !slices.Contains(client.Tags, page.Tag) {
				continue
			}
		}
		if page.Permission != "" {
			if !slices.Contains(client.Permissions, page.Permission) {
				continue
			}
		}
		if page.Identity != "" {
			if client.Credentials.Identity != page.Identity {
				continue
			}
		}
		if len(page.IDs) > 0 {
			if !slices.Contains(page.IDs, client.ID) {
				continue
			}
		}
		clients = append(clients, client)
	}
	clientPage := mgclients.ClientsPage{
		Page: mgclients.Page{
			Total:  total,
			Offset: page.Offset,
			Limit:  page.Limit,
		},
	}

	if len(clients) < int(page.Limit) {
		clientPage.Clients = clients
		return clientPage, nil
	}

	clientPage.Clients = clients[page.Offset : page.Offset+page.Limit]

	return clientPage, nil
}

func (repo *repository) RetrieveAllBasicInfo(ctx context.Context, pm mgclients.Page) (mgclients.ClientsPage, error) {
	clientPage, err := repo.filterUsers(ctx, pm)
	if err != nil {
		return mgclients.ClientsPage{}, err
	}
	for i, client := range clientPage.Clients {
		clientPage.Clients[i] = mgclients.Client{
			ID:        client.ID,
			Name:      client.Name,
			CreatedAt: client.CreatedAt,
			UpdatedAt: client.UpdatedAt,
			Status:    client.Status,
		}
	}

	return clientPage, nil
}

// This is not used by users service also when being used it is used to filter by IDs only not by other fields.
func (repo *repository) RetrieveAllByIDs(ctx context.Context, pm mgclients.Page) (mgclients.ClientsPage, error) {
	identities, resp, err := repo.IdentityAPI.ListIdentities(ctx).Page(int64(pm.Offset)).PerPage(int64(pm.Limit)).IdsFilter(pm.IDs).Execute()
	if err != nil {
		return mgclients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, decodeError(resp))
	}
	total, err := strconv.ParseUint(resp.Header.Get("X-Total-Count"), 10, 64)
	if err != nil {
		return mgclients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	clients := []mgclients.Client{}
	for _, identity := range identities {
		clients = append(clients, toClient(&identity))
	}

	return mgclients.ClientsPage{
		Page: mgclients.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
		Clients: clients,
	}, nil
}

func (repo *repository) Update(ctx context.Context, user mgclients.Client) (mgclients.Client, error) {
	rclient, err := repo.RetrieveByID(ctx, user.ID)
	if err != nil {
		return mgclients.Client{}, err
	}

	identity, resp, err := repo.IdentityAPI.UpdateIdentity(ctx, user.ID).UpdateIdentityBody(ory.UpdateIdentityBody{
		Traits: map[string]interface{}{
			"username": user.Name,
			"email":    rclient.Credentials.Identity,
		},
		MetadataPublic: user.Metadata,
	}).Execute()
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrUpdateEntity, decodeError(resp))
	}

	return toClient(identity), nil
}

func (repo *repository) UpdateTags(ctx context.Context, user mgclients.Client) (mgclients.Client, error) {
	rclient, err := repo.RetrieveByID(ctx, user.ID)
	if err != nil {
		return mgclients.Client{}, err
	}

	identity, resp, err := repo.IdentityAPI.UpdateIdentity(ctx, user.ID).UpdateIdentityBody(ory.UpdateIdentityBody{
		Traits: map[string]interface{}{
			"enterprise": slices.Contains(user.Tags, "enterprise"),
			"newsletter": slices.Contains(user.Tags, "newsletter"),
			"username":   rclient.Name,
			"email":      rclient.Credentials.Identity,
		},
	}).Execute()
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrUpdateEntity, decodeError(resp))
	}

	return toClient(identity), nil
}

func (repo *repository) UpdateIdentity(ctx context.Context, user mgclients.Client) (mgclients.Client, error) {
	rclient, err := repo.RetrieveByID(ctx, user.ID)
	if err != nil {
		return mgclients.Client{}, err
	}

	identity, resp, err := repo.IdentityAPI.UpdateIdentity(ctx, user.ID).UpdateIdentityBody(ory.UpdateIdentityBody{
		Traits: map[string]interface{}{
			"email":    user.Credentials.Identity,
			"username": rclient.Name,
		},
	}).Execute()
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrUpdateEntity, decodeError(resp))
	}

	return toClient(identity), nil
}

func (repo *repository) UpdateSecret(ctx context.Context, user mgclients.Client) (mgclients.Client, error) {
	hashedPassword, err := repo.hasher.Hash(user.Credentials.Secret)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	identity, resp, err := repo.IdentityAPI.UpdateIdentity(ctx, user.ID).UpdateIdentityBody(ory.UpdateIdentityBody{
		Credentials: &ory.IdentityWithCredentials{
			Password: &ory.IdentityWithCredentialsPassword{
				Config: &ory.IdentityWithCredentialsPasswordConfig{
					HashedPassword: &hashedPassword,
					Password:       &user.Credentials.Secret,
				},
			},
		},
	}).Execute()
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrUpdateEntity, decodeError(resp))
	}

	return toClient(identity), nil
}

func (repo *repository) ChangeStatus(ctx context.Context, user mgclients.Client) (mgclients.Client, error) {
	rclient, err := repo.RetrieveByID(ctx, user.ID)
	if err != nil {
		return mgclients.Client{}, err
	}

	identity, resp, err := repo.IdentityAPI.UpdateIdentity(ctx, user.ID).UpdateIdentityBody(ory.UpdateIdentityBody{
		State: mgclients.ToOryState(user.Status),
		Traits: map[string]interface{}{
			"email":    rclient.Credentials.Identity,
			"username": rclient.Name,
		},
	}).Execute()
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrUpdateEntity, decodeError(resp))
	}

	return toClient(identity), nil
}

func (repo *repository) UpdateRole(ctx context.Context, user mgclients.Client) (mgclients.Client, error) {
	rclient, err := repo.RetrieveByID(ctx, user.ID)
	if err != nil {
		return mgclients.Client{}, err
	}

	identity, resp, err := repo.IdentityAPI.UpdateIdentity(ctx, user.ID).UpdateIdentityBody(ory.UpdateIdentityBody{
		MetadataAdmin: map[string]interface{}{
			"role":        user.Role,
			"permissions": rclient.Permissions,
		},
		Traits: map[string]interface{}{
			"email":    rclient.Credentials.Identity,
			"username": rclient.Name,
		},
	}).Execute()
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrUpdateEntity, decodeError(resp))
	}

	return toClient(identity), nil
}

func (repo *repository) CheckSuperAdmin(ctx context.Context, adminID string) error {
	rclient, err := repo.RetrieveByID(ctx, adminID)
	if err != nil {
		return svcerr.ErrAuthorization
	}
	if rclient.Role != mgclients.AdminRole {
		return svcerr.ErrAuthorization
	}
	return nil
}

func decodeError(response *http.Response) error {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}
	slog.Warn("Error response", slog.Any("body", string(body)))

	var content struct {
		Error ory.GenericError `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &content); err != nil {
		return fmt.Errorf("error unmarshalling response body: %w", err)
	}

	return errors.New(content.Error.Message)
}

func toClient(identity *ory.Identity) mgclients.Client {
	tags := []string{}
	if identity.Traits.(map[string]interface{})["enterprise"] != nil {
		if identity.Traits.(map[string]interface{})["enterprise"].(bool) {
			tags = append(tags, "enterprise")
		}
	}
	if identity.Traits.(map[string]interface{})["newsletter"] != nil {
		if identity.Traits.(map[string]interface{})["newsletter"].(bool) {
			tags = append(tags, "newsletter")
		}
	}

	username := ""
	if identity.Traits.(map[string]interface{})["username"] != nil {
		username = identity.Traits.(map[string]interface{})["username"].(string)
	}

	email := ""
	if identity.Traits.(map[string]interface{})["email"] != nil {
		email = identity.Traits.(map[string]interface{})["email"].(string)
	}

	status := mgclients.EnabledStatus
	if *identity.State == ory.IDENTITYSTATE_INACTIVE {
		status = mgclients.DisabledStatus
	}

	role := mgclients.UserRole
	if identity.MetadataAdmin != nil {
		if identity.MetadataAdmin["role"] != nil {
			var err error
			role, err = mgclients.ToRole(identity.MetadataAdmin["role"].(string))
			if err != nil {
				slog.Warn("Invalid role", slog.Any("role", identity.MetadataAdmin["role"]))
			}
		}
	}

	permissions := []string{}
	if identity.MetadataAdmin != nil {
		if identity.MetadataAdmin["permissions"] != nil {
			for _, p := range identity.MetadataAdmin["permissions"].([]interface{}) {
				permissions = append(permissions, p.(string))
			}
		}
	}

	return mgclients.Client{
		ID:        identity.Id,
		Name:      username,
		Tags:      tags,
		CreatedAt: *identity.CreatedAt,
		UpdatedAt: *identity.UpdatedAt,
		Metadata:  identity.MetadataPublic,
		Credentials: mgclients.Credentials{
			Identity: email,
		},
		Role:        role,
		Status:      status,
		Permissions: permissions,
	}
}
