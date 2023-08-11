package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/auth/spicedb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defSpiceDBHost       = "localhost"
	defSpiceDBPort       = "50051"
	defSpicePreSharedKey = "12345678"
	defSpiceDBSchemaFile = "./docker/spicedb/schema.zed"
)

func main() {
	spicedbClient, err := initSpiceDB()
	if err != nil {
		log.Fatalf("failed to init spicedb grpc client : %s\n", err.Error())
	}

	if err := initSchema(spicedbClient, defSpiceDBSchemaFile); err != nil {
		log.Fatalln(err)
	}

	pa := spicedb.NewPolicyAgent(spicedbClient)

	_ = pa

	// Operation1(pa)
	// Operation4(pa)

	// Operation9(spicedbClient)
	// Operation9(spicedbClient)
	// Operation9(spicedbClient)
	// Operation9(spicedbClient)
	// Operation9(spicedbClient)
	// Operation11(spicedbClient)
	// Operation11(spicedbClient)
	Operation10(pa)

	// Operation4(pa)
	// fmt.Println(pa.AddPolicy(context.Background(), auth.PolicyReq{
	// 	SubjectType: "user",
	// 	Subject:     "_any_body",
	// 	Relation:    "create",
	// 	Permission:  "",
	// 	ObjectType:  "organization",
	// 	Object:      "mainflux",
	// }))
}
func Operation10(pa auth.PolicyAgent) {
	startTime := time.Now()
	res, _ := pa.RetrieveAllObjects(context.TODO(), auth.PolicyReq{
		ObjectType:  "thing",
		Permission:  "view",
		SubjectType: "user",
		Subject:     "eec2819d-430b-400c-8704-c0da1f617d51",
	})
	fmt.Println(len(res))
	fmt.Println(time.Since(startTime))

}
func Operation11(ac *authzed.Client) {

	respStream, err := ac.PermissionsServiceClient.LookupResources(context.TODO(), &v1.LookupResourcesRequest{
		ResourceObjectType: "thing",
		Permission:         "view",
		Subject:            &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: "user", ObjectId: "f8ab7ae9-f33d-4d28-a8c8-6f85637e0999"}},
	})

	if err != nil {
		fmt.Println("failed send request")
		panic(err)
	}
	startTime := time.Now()
	for {
		resp, err := respStream.Recv()
		_ = resp
		switch {
		case errors.Is(err, io.EOF):
			fmt.Println(time.Since(startTime))
			return
		case err != nil:
			fmt.Println(time.Since(startTime))
			fmt.Println("failed stream request")
			panic(err)
		default:
		}
	}
}
func Operation9(ac *authzed.Client) {

	respStream, err := ac.PermissionsServiceClient.LookupResources(context.TODO(), &v1.LookupResourcesRequest{
		ResourceObjectType: "thing",
		Permission:         "view",
		Subject:            &v1.SubjectReference{Object: &v1.ObjectReference{ObjectType: "user", ObjectId: "f8ab7ae9-f33d-4d28-a8c8-6f85637e0999"}},
	})

	if err != nil {
		fmt.Println("failed send request")
		panic(err)
	}
	startTime := time.Now()
	for {
		resp, err := respStream.Recv()
		_ = resp
		switch {
		case errors.Is(err, io.EOF):
			fmt.Println(time.Since(startTime))
			return
		case err != nil:
			fmt.Println(time.Since(startTime))
			fmt.Println("failed stream request")
			panic(err)
		default:
		}
	}
}
func Operation8(pa auth.PolicyAgent) {
	err := pa.DeletePolicies(context.Background(), []auth.PolicyReq{{
		Subject:     "randone",
		SubjectType: "user",
		Relation:    "viewer",
		Object:      "group_1",
		ObjectType:  "group",
	}, {
		Subject:     "group_1",
		SubjectType: "group",
		Relation:    "group",
		Object:      "randone",
		ObjectType:  "thing",
	}})
	fmt.Println(err)
}
func Operation7(pa auth.PolicyAgent) {
	pl, npt, err := pa.RetrieveSubjects(context.Background(), auth.PolicyReq{ObjectType: "group", Object: "group_4", Permission: "parent_group", SubjectType: "group"}, "", 1000)
	fmt.Println(pl, npt, err)
}
func Operation6(pa auth.PolicyAgent) {
	prs := []auth.PolicyReq{}
	for i := 3; i <= 3; i++ {
		for j := 1; j <= 100; j++ {
			prs = append(prs, auth.PolicyReq{
				Namespace:       "",
				Subject:         fmt.Sprintf("user_%d", i),
				SubjectType:     "user",
				SubjectRelation: "",
				Object:          fmt.Sprintf("thing_%d", j),
				ObjectType:      "thing",
				Relation:        "owner",
				Permission:      "",
			})
		}
	}
	err := pa.AddPolicies(context.Background(), prs)
	fmt.Println(err)

}

func Operation5(pa auth.PolicyAgent) {
	policies, err := pa.RetrieveAllSubjects(context.Background(), auth.PolicyReq{
		SubjectType: "user",
		ObjectType:  "thing",
		Object:      "thing_1",
		Permission:  "view",
	})

	if err != nil {
		fmt.Println("failed to list policies", err)
	}

	for _, policy := range policies {
		fmt.Println(policy)
	}

}
func Operation4(pa auth.PolicyAgent) {
	policies, err := pa.RetrieveAllObjects(context.Background(), auth.PolicyReq{
		Subject:         "user_1",
		SubjectType:     "user",
		SubjectRelation: "",
		ObjectType:      "thing",
		Permission:      "edit",
	})

	if err != nil {
		fmt.Println("failed to list policies", err)
	}

	for _, policy := range policies {
		fmt.Println(policy)
	}
}

func Operation3(pa auth.PolicyAgent) {
	for j := 1; j <= 100; j++ {

		err := pa.AddPolicy(context.Background(), auth.PolicyReq{
			Namespace:       "",
			Subject:         "group_2",
			SubjectType:     "group",
			SubjectRelation: "",
			Object:          fmt.Sprintf("thing_%d", j),
			ObjectType:      "thing",
			Relation:        "group",
			Permission:      "",
		})
		if err != nil {
			fmt.Println(err)
		}
	}
}
func Operation2(pa auth.PolicyAgent) {
	policies, err := pa.RetrieveAllSubjects(context.Background(), auth.PolicyReq{
		Subject:     "user_1",
		SubjectType: "user",
		ObjectType:  "thing",
		Permission:  "delete",
	})

	if err != nil {
		fmt.Println("failed to list policies", err)
	}

	for _, policy := range policies {
		fmt.Println(policy)
	}
}

func Operation1(pa auth.PolicyAgent) {
	for i := 1; i <= 2; i++ {
		for j := 1; j <= 100; j++ {

			err := pa.AddPolicy(context.Background(), auth.PolicyReq{
				Namespace:       "",
				Subject:         fmt.Sprintf("user_%d", i),
				SubjectType:     "user",
				SubjectRelation: "",
				Object:          fmt.Sprintf("thing_%d", j),
				ObjectType:      "thing",
				Relation:        "owner",
				Permission:      "",
			})
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}
func initSpiceDB() (*authzed.Client, error) {
	return authzed.NewClient(
		fmt.Sprintf("%s:%s", defSpiceDBHost, defSpiceDBPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(defSpicePreSharedKey),
	)
}

func initSchema(client *authzed.Client, schemaFilePath string) error {
	schemaContent, err := os.ReadFile(schemaFilePath)

	if err != nil {
		return fmt.Errorf("failed to read spice db schema file : %w", err)
	}
	_, err = client.SchemaServiceClient.WriteSchema(context.Background(), &v1.WriteSchemaRequest{Schema: string(schemaContent)})
	if err != nil {
		return fmt.Errorf("failed to create schema in spicedb : %w", err)
	}
	return nil
}
