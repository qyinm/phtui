package dto

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/qyinm/phtui/types"
)

func TestDTOJSONMarshal(t *testing.T) {
	product := types.NewProduct(
		"Demo",
		"Fast demos",
		[]string{"Developer Tools"},
		42,
		7,
		"demo",
		"https://img.example/demo.png",
		1,
	)
	detail := types.NewProductDetail(
		product,
		"Detailed description",
		4.8,
		12,
		200,
		"Maker says hi",
		"https://demo.example",
		[]string{"Developer Tools"},
		[]string{"https://x.com/demo"},
		time.Date(2026, 2, 26, 9, 0, 0, 0, time.UTC),
		"Maker",
		"https://producthunt.com/@maker",
		[]types.ProConTag{
			types.NewProConTag("Fast", "Positive", 5),
			types.NewProConTag("Expensive", "Negative", 2),
		},
		"$20/month",
	)

	productDTO := FromProduct(product)
	detailDTO := FromProductDetail(detail)
	categoryDTO := FromCategory(types.NewCategoryLink("AI Agents", "ai-agents"))

	if _, err := json.Marshal(productDTO); err != nil {
		t.Fatalf("marshal product dto: %v", err)
	}
	b, err := json.Marshal(detailDTO)
	if err != nil {
		t.Fatalf("marshal detail dto: %v", err)
	}
	if _, err := json.Marshal(categoryDTO); err != nil {
		t.Fatalf("marshal category dto: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal detail dto: %v", err)
	}

	if got["slug"] != "demo" {
		t.Fatalf("unexpected slug: %v", got["slug"])
	}
	if got["pricing_type"] != "paid" {
		t.Fatalf("unexpected pricing_type: %v", got["pricing_type"])
	}
	if got["pricing_amount"] != "$20" {
		t.Fatalf("unexpected pricing_amount: %v", got["pricing_amount"])
	}
	if got["pricing_period"] != "month" {
		t.Fatalf("unexpected pricing_period: %v", got["pricing_period"])
	}
	if got["launch_date"] != "2026-02-26" {
		t.Fatalf("unexpected launch_date: %v", got["launch_date"])
	}
}

func TestDTOFields(t *testing.T) {
	assertNoInterfaceFields(t, reflect.TypeOf(Product{}))
	assertNoInterfaceFields(t, reflect.TypeOf(ProductDetail{}))
	assertNoInterfaceFields(t, reflect.TypeOf(Category{}))
	assertNoInterfaceFields(t, reflect.TypeOf(ProCon{}))
}

func assertNoInterfaceFields(t *testing.T, typ reflect.Type) {
	t.Helper()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldType := field.Type

		if field.Anonymous {
			assertNoInterfaceFields(t, fieldType)
			continue
		}

		switch fieldType.Kind() {
		case reflect.Interface:
			t.Fatalf("field %s in %s must not be interface type", field.Name, typ.Name())
		case reflect.Struct:
			assertNoInterfaceFields(t, fieldType)
		case reflect.Slice, reflect.Array:
			if fieldType.Elem().Kind() == reflect.Interface {
				t.Fatalf("field %s in %s must not contain interface elements", field.Name, typ.Name())
			}
			if fieldType.Elem().Kind() == reflect.Struct {
				assertNoInterfaceFields(t, fieldType.Elem())
			}
		}
	}
}
