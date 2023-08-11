package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/mainflux/mainflux/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	fileName = "Things1MRelation.json"

	incCount = 1000

	defSpiceDBHost       = "localhost"
	defSpiceDBPort       = "50051"
	defSpicePreSharedKey = "12345678"
	defSpiceDBSchemaFile = "./docker/spicedb/schema.zed"
)

func main() {
	f, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("failed to open file %s : %s \n", fileName, err.Error())
	}

	prs := []auth.PolicyReq{}
	if err := json.NewDecoder(f).Decode(&prs); err != nil {
		log.Fatalf("failed to decode file %s : %s \n", fileName, err.Error())
	}

	count := len(prs)
	fmt.Println("Number of Policies : ", count)

	spicedbClient, err := initSpiceDB()
	if err != nil {
		log.Fatalf("failed to init spicedb grpc client : %s\n", err.Error())
	}

	for i := 0; i <= count; i += incCount {
		tempPrs := prs[i : i+incCount]
		tempCount := len(tempPrs)
		updates := []*v1.RelationshipUpdate{}
		for _, pr := range tempPrs {
			updates = append(updates, &v1.RelationshipUpdate{
				Operation: v1.RelationshipUpdate_OPERATION_CREATE,
				Relationship: &v1.Relationship{
					Resource: &v1.ObjectReference{ObjectType: pr.ObjectType, ObjectId: pr.Object},
					Relation: pr.Relation,
					Subject:  &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: pr.SubjectType, ObjectId: pr.Subject}, OptionalRelation: pr.SubjectRelation},
				},
			})
		}
		if len(updates) > 0 {
			resp, err := spicedbClient.PermissionsServiceClient.WriteRelationships(context.Background(), &v1.WriteRelationshipsRequest{Updates: updates})
			if err != nil {
				log.Fatalf("failed to add policies : %s\n", err.Error())
			}
			fmt.Println()
			fmt.Printf("Successfully added %d policies, token : %s \n", tempCount, resp.WrittenAt.Token)
		}
	}

}

func initSpiceDB() (*authzed.Client, error) {
	return authzed.NewClient(
		fmt.Sprintf("%s:%s", defSpiceDBHost, defSpiceDBPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(512000000), grpc.MaxCallRecvMsgSize(512000000)),
		grpc.WithReadBufferSize(512000000),
		grpc.WithWriteBufferSize(512000000),
		grpcutil.WithInsecureBearerToken(defSpicePreSharedKey),
	)
}
