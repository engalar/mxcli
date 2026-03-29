// SPDX-License-Identifier: Apache-2.0

package mpr

import (
	"testing"
)

const testAsyncAPIYAML = `asyncapi: 2.2.0
info:
  title: "ShopEventsSvc"
  version: "1.0.0"
  description: "Shop events for order processing"
channels:
  c79d2901578f4ddab69688bde6eaf98c:
    subscribe:
      operationId: receiveOrderChangedEventEvents
      message:
        $ref: '#/components/messages/OrderChangedEvent'
  abc123:
    publish:
      operationId: sendProductUpdatedEvents
      message:
        $ref: '#/components/messages/ProductUpdated'
components:
  messages:
    OrderChangedEvent:
      name: OrderChangedEvent
      title: OrderChangedEvent event
      description: "Fired when an order changes"
      contentType: application/json
      payload:
        $ref: '#/components/schemas/OrderChangedEventPayload'
    ProductUpdated:
      name: ProductUpdated
      title: Product Updated
      description: ""
      contentType: application/json
      payload:
        $ref: '#/components/schemas/ProductUpdatedPayload'
  schemas:
    OrderChangedEventPayload:
      type: object
      properties:
        OrderId:
          type: integer
          format: int64
        CustomerId:
          type: integer
          format: int64
    ProductUpdatedPayload:
      type: object
      properties:
        ProductName:
          type: string
        Price:
          type: number
          format: double
        InStock:
          type: boolean
defaultContentType: application/json
`

func TestParseAsyncAPI(t *testing.T) {
	doc, err := ParseAsyncAPI(testAsyncAPIYAML)
	if err != nil {
		t.Fatalf("ParseAsyncAPI failed: %v", err)
	}

	if doc.Version != "2.2.0" {
		t.Errorf("expected version 2.2.0, got %s", doc.Version)
	}
	if doc.Title != "ShopEventsSvc" {
		t.Errorf("expected title ShopEventsSvc, got %s", doc.Title)
	}
	if doc.Description != "Shop events for order processing" {
		t.Errorf("expected description, got %q", doc.Description)
	}

	// Check channels
	if len(doc.Channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(doc.Channels))
	}

	var subChannel, pubChannel *AsyncAPIChannel
	for _, ch := range doc.Channels {
		if ch.OperationType == "subscribe" {
			subChannel = ch
		} else if ch.OperationType == "publish" {
			pubChannel = ch
		}
	}

	if subChannel == nil {
		t.Fatal("subscribe channel not found")
	}
	if subChannel.MessageRef != "OrderChangedEvent" {
		t.Errorf("expected message ref OrderChangedEvent, got %s", subChannel.MessageRef)
	}
	if subChannel.OperationID != "receiveOrderChangedEventEvents" {
		t.Errorf("expected operationId receiveOrderChangedEventEvents, got %s", subChannel.OperationID)
	}

	if pubChannel == nil {
		t.Fatal("publish channel not found")
	}
	if pubChannel.MessageRef != "ProductUpdated" {
		t.Errorf("expected message ref ProductUpdated, got %s", pubChannel.MessageRef)
	}

	// Check messages
	if len(doc.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(doc.Messages))
	}

	orderMsg := doc.FindMessage("OrderChangedEvent")
	if orderMsg == nil {
		t.Fatal("OrderChangedEvent message not found")
	}
	if orderMsg.Description != "Fired when an order changes" {
		t.Errorf("expected description, got %q", orderMsg.Description)
	}
	if len(orderMsg.Properties) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(orderMsg.Properties))
	}

	// Check property resolution
	var orderIdProp *AsyncAPIProperty
	for _, p := range orderMsg.Properties {
		if p.Name == "OrderId" {
			orderIdProp = p
			break
		}
	}
	if orderIdProp == nil {
		t.Fatal("OrderId property not found")
	}
	if orderIdProp.Type != "integer" {
		t.Errorf("expected type integer, got %s", orderIdProp.Type)
	}
	if orderIdProp.Format != "int64" {
		t.Errorf("expected format int64, got %s", orderIdProp.Format)
	}

	// Check ProductUpdated message
	prodMsg := doc.FindMessage("ProductUpdated")
	if prodMsg == nil {
		t.Fatal("ProductUpdated message not found")
	}
	if len(prodMsg.Properties) != 3 {
		t.Fatalf("expected 3 properties, got %d", len(prodMsg.Properties))
	}
}

func TestParseAsyncAPIEmpty(t *testing.T) {
	_, err := ParseAsyncAPI("")
	if err == nil {
		t.Error("expected error for empty document")
	}
}
