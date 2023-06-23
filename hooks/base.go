package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/types"
)

// "github.com/pocketbase/dbx"

type Hooks struct {
	ID        int           `db:"id" json:"id"`
	TableName string        `db:"tableName" id:"tableName"`
	URL       string        `db:"url" id:"url"`
	Op        string        `db:"op" id:"op"`
	Header    types.JsonMap `db:"header" id:"header"`
}

func New() *pocketbase.PocketBase {
	app := pocketbase.New()

	app.OnAfterBootstrap().Add(func(e *core.BootstrapEvent) error {

		createModels(app)
		initModelEvents(app)

		return nil
	})

	return app
}

func createModels(app *pocketbase.PocketBase) error {

	// hooks := make(map[string]string)
	// hooks["id"] = "int PRIMARY KEY AUTO"
	// hooks["TableName"] = "TEXT DEFAULT '' NOT NULL"
	// hooks["url"] = "TEXT DEFAULT '' NOT NULL"
	// hooks["action"] = "TEXT DEFAULT ''"
	// app.Dao().DB().CreateTable("hooks", hooks)

	collection := &models.Collection{
		Name:       "hooks",
		Type:       models.CollectionTypeBase,
		ListRule:   nil,
		ViewRule:   types.Pointer("@request.auth.id != '' && @request.auth.role = 'superUser'"),
		CreateRule: types.Pointer("@request.auth.id != '' && @request.auth.role = 'superUser'"),
		DeleteRule: types.Pointer("@request.auth.id != '' && @request.auth.role = 'superUser'"),
		Schema: schema.NewSchema(
			&schema.SchemaField{
				Name:     "id",
				Type:     schema.FieldTypeText,
				Required: true,
			},
			&schema.SchemaField{
				Name:     "tableName",
				Type:     schema.FieldTypeText,
				Required: true,
			},
			&schema.SchemaField{
				Name:     "url",
				Type:     schema.FieldTypeUrl,
				Required: true,
			},
			&schema.SchemaField{
				Name:     "op",
				Type:     schema.FieldTypeSelect,
				Options:  &schema.SelectOptions{MaxSelect: 1, Values: []string{"insert", "update", "delete"}},
				Required: true,
			},
			&schema.SchemaField{
				Name:     "header",
				Type:     schema.FieldTypeJson,
				Required: false,
			},
			// ,
			// &schema.SchemaField{
			// 	Name:     "user",
			// 	Type:     schema.FieldTypeRelation,
			// 	Required: true,
			// 	Options:  &schema.RelationOptions{
			// 		MaxSelect:     types.Pointer(1),
			// 		CollectionId:  "ae40239d2bc4477",
			// 		CascadeDelete: true,
			// 	},
			// },
		),
	}

	if err := app.Dao().SaveCollection(collection); err != nil {
		return err
	}

	return nil
}

func runHook(app *pocketbase.PocketBase, e *models.Record, collectionName string, eventType string) error {

	now := time.Now()
	recordBytes, err := json.Marshal(e)
	if err != nil {
		// handle the error
		return err
	}

	obj := map[string]interface{}{
		"op":     eventType,
		"record": recordBytes,
		"time":   now.Unix(),
	}

	jsonData, err := json.Marshal(obj)
	if err != nil {
		// handle the error
		return err
	}

	records, err := app.Dao().FindRecordsByExpr("hooks", dbx.HashExp{"tableName": collectionName, "action": eventType})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", records[0].GetString("url"), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	// Set appropriate headers, if needed
	req.Header.Set("Content-Type", "application/json")

	// Create an HTTP client and send the request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Handle the response as needed

	return nil
}

func initModelEvents(app *pocketbase.PocketBase) {

	// app.OnRecordBeforeCreateRequest().Add(func(e *core.RecordCreateEvent) error {
	// 	return nil
	// })

	app.OnRecordAfterCreateRequest().Add(func(e *core.RecordCreateEvent) error {
		runHook(app, e.Record, e.Collection.Name, "insert")
		return nil
	})

	// app.OnRecordBeforeUpdateRequest().Add(func(e *core.RecordUpdateEvent) error {
	// 	log.Println(e.Record.GetString("title")) // not saved yet
	// 	return nil
	// })

	app.OnRecordAfterUpdateRequest().Add(func(e *core.RecordUpdateEvent) error {
		runHook(app, e.Record, e.Collection.Name, "update")
		return nil
	})

	// app.OnRecordBeforeDeleteRequest().Add(func(e *core.RecordDeleteEvent) error {
	// 	log.Println(e.Record.Id) // not deleted yet
	// 	return nil
	// })

	app.OnRecordAfterDeleteRequest().Add(func(e *core.RecordDeleteEvent) error {
		runHook(app, e.Record, e.Collection.Name, "delete")
		return nil
	})

}
