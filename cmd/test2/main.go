package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/uuid"
)

const sqlTemplate = "INSERT INTO public.things (id, \"owner\", \"key\", \"name\", metadata) VALUES('%s'::uuid, '%s', '%s', '%s', '{}'::jsonb);\n"
const csvTemplate = "\"%s\",\"%s\",\"%s\",\"%s\",\"%s\"\n"
const userEmail = "user1@example.com"
const userId = "f8ab7ae9-f33d-4d28-a8c8-6f85637e0999"

// 1B = 1,000,000,000 | 100C = 100,00,00,000
// 100M = 100,000,000 | 10C = 10,00,00,000
// 10M = 10,000,000 | 1C = 1,00,00,000
// 1M = 1,000,000  | 10L = 10,00,000
// 100K = 100,000 | 1L = 1,00,000
// 10K = 10,000 | 10K = 10,000
// 1K = 1,000 | 1K = 1,000
const limit = 1000000

func main() {
	runId := rand.Intn(100000)
	uuidGen := uuid.New()
	startTime := time.Now()
	stringBuffer := bytes.NewBufferString("")
	prs := []auth.PolicyReq{}
	for i := 1; i <= limit; i++ {
		id, _ := uuidGen.ID()
		key, _ := uuidGen.ID()
		name := fmt.Sprintf("thing_%d_%d", i, runId)
		// sql := fmt.Sprintf(sqlTemplate, id, userEmail, key, name)
		// stringBuffer.WriteString(sql)
		csv := fmt.Sprintf(csvTemplate, id, userEmail, key, name, "{}")
		stringBuffer.WriteString(csv)
		prs = append(prs, auth.PolicyReq{
			ObjectType:  "thing",
			Object:      id,
			Relation:    "owner",
			Subject:     userId,
			SubjectType: "user",
		})
	}
	// dumpSQL(stringBuffer)
	dumpCSV(stringBuffer)
	dumpRelation(prs)
	fmt.Println("Elapsed time ", time.Since(startTime))
}

func dumpRelation(prs []auth.PolicyReq) {
	f, err := os.Create("Things1MRelation.json")
	if err != nil {
		log.Fatal("Couldn't open file")
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(prs)
	if err != nil {
		log.Fatal("Write binary failed", err)
	}
}

func dumpSQL(buf *bytes.Buffer) {
	f, err := os.Create("Things1M.sql")
	if err != nil {
		log.Fatalf("Couldn't open file : %s", err.Error())
	}
	defer f.Close()
	_, err = f.Write(buf.Bytes())
	if err != nil {
		log.Fatalf("Unable to write : %s", err.Error())
	}
}

func dumpCSV(buf *bytes.Buffer) {
	f, err := os.Create("Things1M.csv")
	if err != nil {
		log.Fatalf("Couldn't open file : %s", err.Error())
	}
	defer f.Close()
	_, err = f.Write(buf.Bytes())
	if err != nil {
		log.Fatalf("Unable to write : %s", err.Error())
	}
}
