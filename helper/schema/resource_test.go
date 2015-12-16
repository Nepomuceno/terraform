package schema

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/lang/ast"
	"github.com/hashicorp/terraform/terraform"
)

func TestResourceApply_create(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	called := false
	r.Create = func(d *ResourceData, m interface{}) error {
		called = true
		d.SetId("foo")
		return nil
	}

	var s *terraform.InstanceState = nil

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": &terraform.ResourceAttrDiff{
				New: "42",
			},
		},
	}

	actual, err := r.Apply(s, d, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !called {
		t.Fatal("not called")
	}

	expected := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":  "foo",
			"foo": "42",
		},
		Meta: map[string]string{
			"schema_version": "2",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceApply_destroy(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	called := false
	r.Delete = func(d *ResourceData, m interface{}) error {
		called = true
		return nil
	}

	s := &terraform.InstanceState{
		ID: "bar",
	}

	d := &terraform.InstanceDiff{
		Destroy: true,
	}

	actual, err := r.Apply(s, d, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !called {
		t.Fatal("delete not called")
	}

	if actual != nil {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceApply_destroyCreate(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},

			"tags": &Schema{
				Type:     TypeMap,
				Optional: true,
				Computed: true,
			},
		},
	}

	change := false
	r.Create = func(d *ResourceData, m interface{}) error {
		change = d.HasChange("tags")
		d.SetId("foo")
		return nil
	}
	r.Delete = func(d *ResourceData, m interface{}) error {
		return nil
	}

	var s *terraform.InstanceState = &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo":       "bar",
			"tags.Name": "foo",
		},
	}

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": &terraform.ResourceAttrDiff{
				New:         "42",
				RequiresNew: true,
			},
			"tags.Name": &terraform.ResourceAttrDiff{
				Old:         "foo",
				New:         "foo",
				RequiresNew: true,
			},
		},
	}

	actual, err := r.Apply(s, d, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !change {
		t.Fatal("should have change")
	}

	expected := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":        "foo",
			"foo":       "42",
			"tags.#":    "1",
			"tags.Name": "foo",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceApply_destroyPartial(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
		SchemaVersion: 3,
	}

	r.Delete = func(d *ResourceData, m interface{}) error {
		d.Set("foo", 42)
		return fmt.Errorf("some error")
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	d := &terraform.InstanceDiff{
		Destroy: true,
	}

	actual, err := r.Apply(s, d, nil)
	if err == nil {
		t.Fatal("should error")
	}

	expected := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"id":  "bar",
			"foo": "42",
		},
		Meta: map[string]string{
			"schema_version": "3",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected:\n%#v\n\ngot:\n%#v", expected, actual)
	}
}

func TestResourceApply_update(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Update = func(d *ResourceData, m interface{}) error {
		d.Set("foo", 42)
		return nil
	}

	s := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": &terraform.ResourceAttrDiff{
				New: "13",
			},
		},
	}

	actual, err := r.Apply(s, d, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"id":  "foo",
			"foo": "42",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceApply_updateNoCallback(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Update = nil

	s := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	d := &terraform.InstanceDiff{
		Attributes: map[string]*terraform.ResourceAttrDiff{
			"foo": &terraform.ResourceAttrDiff{
				New: "13",
			},
		},
	}

	actual, err := r.Apply(s, d, nil)
	if err == nil {
		t.Fatal("should error")
	}

	expected := &terraform.InstanceState{
		ID: "foo",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceInternalValidate(t *testing.T) {
	cases := []struct {
		In  *Resource
		Err bool
	}{
		{
			nil,
			true,
		},

		// No optional and no required
		{
			&Resource{
				Schema: map[string]*Schema{
					"foo": &Schema{
						Type:     TypeInt,
						Optional: true,
						Required: true,
					},
				},
			},
			true,
		},

		// Update undefined for non-ForceNew field
		{
			&Resource{
				Create: func(d *ResourceData, meta interface{}) error { return nil },
				Schema: map[string]*Schema{
					"boo": &Schema{
						Type:     TypeInt,
						Optional: true,
					},
				},
			},
			true,
		},

		// Update defined for ForceNew field
		{
			&Resource{
				Create: func(d *ResourceData, meta interface{}) error { return nil },
				Update: func(d *ResourceData, meta interface{}) error { return nil },
				Schema: map[string]*Schema{
					"goo": &Schema{
						Type:     TypeInt,
						Optional: true,
						ForceNew: true,
					},
				},
			},
			true,
		},
	}

	for i, tc := range cases {
		err := tc.In.InternalValidate(schemaMap{})
		if err != nil != tc.Err {
			t.Fatalf("%d: bad: %s", i, err)
		}
	}
}

func TestResourceRefresh(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		if m != 42 {
			return fmt.Errorf("meta not passed")
		}

		return d.Set("foo", d.Get("foo").(int)+1)
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	expected := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"id":  "bar",
			"foo": "13",
		},
		Meta: map[string]string{
			"schema_version": "2",
		},
	}

	actual, err := r.Refresh(s, 42)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceRefresh_blankId(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		d.SetId("foo")
		return nil
	}

	s := &terraform.InstanceState{
		ID:         "",
		Attributes: map[string]string{},
	}

	actual, err := r.Refresh(s, 42)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != nil {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceRefresh_delete(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		d.SetId("")
		return nil
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	actual, err := r.Refresh(s, 42)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if actual != nil {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceRefresh_existsError(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Exists = func(*ResourceData, interface{}) (bool, error) {
		return false, fmt.Errorf("error")
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		panic("shouldn't be called")
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	actual, err := r.Refresh(s, 42)
	if err == nil {
		t.Fatalf("should error")
	}
	if !reflect.DeepEqual(actual, s) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestResourceRefresh_noExists(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Exists = func(*ResourceData, interface{}) (bool, error) {
		return false, nil
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		panic("shouldn't be called")
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"foo": "12",
		},
	}

	actual, err := r.Refresh(s, 42)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if actual != nil {
		t.Fatalf("should have no state")
	}
}

func TestResourceRefresh_needsMigration(t *testing.T) {
	// Schema v2 it deals only in newfoo, which tracks foo as an int
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"newfoo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		return d.Set("newfoo", d.Get("newfoo").(int)+1)
	}

	r.MigrateState = func(
		v int,
		s *terraform.InstanceState,
		meta interface{}) (*terraform.InstanceState, error) {
		// Real state migration functions will probably switch on this value,
		// but we'll just assert on it for now.
		if v != 1 {
			t.Fatalf("Expected StateSchemaVersion to be 1, got %d", v)
		}

		if meta != 42 {
			t.Fatal("Expected meta to be passed through to the migration function")
		}

		oldfoo, err := strconv.ParseFloat(s.Attributes["oldfoo"], 64)
		if err != nil {
			t.Fatalf("err: %#v", err)
		}
		s.Attributes["newfoo"] = strconv.Itoa(int(oldfoo * 10))
		delete(s.Attributes, "oldfoo")

		return s, nil
	}

	// State is v1 and deals in oldfoo, which tracked foo as a float at 1/10th
	// the scale of newfoo
	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"oldfoo": "1.2",
		},
		Meta: map[string]string{
			"schema_version": "1",
		},
	}

	actual, err := r.Refresh(s, 42)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"id":     "bar",
			"newfoo": "13",
		},
		Meta: map[string]string{
			"schema_version": "2",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad:\n\nexpected: %#v\ngot: %#v", expected, actual)
	}
}

func TestResourceRefresh_noMigrationNeeded(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"newfoo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		return d.Set("newfoo", d.Get("newfoo").(int)+1)
	}

	r.MigrateState = func(
		v int,
		s *terraform.InstanceState,
		meta interface{}) (*terraform.InstanceState, error) {
		t.Fatal("Migrate function shouldn't be called!")
		return nil, nil
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"newfoo": "12",
		},
		Meta: map[string]string{
			"schema_version": "2",
		},
	}

	actual, err := r.Refresh(s, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"id":     "bar",
			"newfoo": "13",
		},
		Meta: map[string]string{
			"schema_version": "2",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad:\n\nexpected: %#v\ngot: %#v", expected, actual)
	}
}

func TestResourceRefresh_stateSchemaVersionUnset(t *testing.T) {
	r := &Resource{
		// Version 1 > Version 0
		SchemaVersion: 1,
		Schema: map[string]*Schema{
			"newfoo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		return d.Set("newfoo", d.Get("newfoo").(int)+1)
	}

	r.MigrateState = func(
		v int,
		s *terraform.InstanceState,
		meta interface{}) (*terraform.InstanceState, error) {
		s.Attributes["newfoo"] = s.Attributes["oldfoo"]
		return s, nil
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"oldfoo": "12",
		},
	}

	actual, err := r.Refresh(s, nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"id":     "bar",
			"newfoo": "13",
		},
		Meta: map[string]string{
			"schema_version": "1",
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad:\n\nexpected: %#v\ngot: %#v", expected, actual)
	}
}

func TestResourceRefresh_migrateStateErr(t *testing.T) {
	r := &Resource{
		SchemaVersion: 2,
		Schema: map[string]*Schema{
			"newfoo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
	}

	r.Read = func(d *ResourceData, m interface{}) error {
		t.Fatal("Read should never be called!")
		return nil
	}

	r.MigrateState = func(
		v int,
		s *terraform.InstanceState,
		meta interface{}) (*terraform.InstanceState, error) {
		return s, fmt.Errorf("triggering an error")
	}

	s := &terraform.InstanceState{
		ID: "bar",
		Attributes: map[string]string{
			"oldfoo": "12",
		},
	}

	_, err := r.Refresh(s, nil)
	if err == nil {
		t.Fatal("expected error, but got none!")
	}
}

func TestResourceValidate(t *testing.T) {
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
			"bar": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
		ValidateFunc: func(g ResourceConfigGetter) (ws []string, es []error) {
			foo, fooKnown := g.Get("foo")
			bar, barKnown := g.Get("bar")

			// Cannot perform validation unless both foo and bar are known
			if !fooKnown || !barKnown {
				return
			}

			if foo.(int) > bar.(int) {
				es = append(es, fmt.Errorf("Foo cannot be greater than bar!"))
			}

			if foo.(int) == bar.(int) {
				ws = append(ws, fmt.Sprintf("Foo is almost greater than bar!"))
			}

			return
		},
	}

	cases := map[string]struct {
		Config           map[string]interface{}
		ExpectedErrors   []error
		ExpectedWarnings []string
		Vars             map[string]ast.Variable
	}{
		"populates errors": {
			Config: map[string]interface{}{
				"foo": 5,
				"bar": 3,
			},
			ExpectedErrors: []error{errors.New("Foo cannot be greater than bar!")},
		},
		"populates warnings": {
			Config: map[string]interface{}{
				"foo": 3,
				"bar": 3,
			},
			ExpectedWarnings: []string{"Foo is almost greater than bar!"},
		},
		"can pass": {
			Config: map[string]interface{}{
				"foo": 3,
				"bar": 4,
			},
		},
		"validatefuncs can check for known/unknown": {
			Config: map[string]interface{}{
				"foo": "${var.notset}",
				"bar": 4,
			},
			Vars: map[string]ast.Variable{
				"var.notset": ast.Variable{
					Value: config.UnknownVariableValue,
					Type:  ast.TypeString,
				},
			},
		},
	}

	for _, tc := range cases {
		rawConfig, err := config.NewRawConfig(tc.Config)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		if err := rawConfig.Interpolate(tc.Vars); err != nil {
			t.Fatalf("err: %s", err)
		}
		config := terraform.NewResourceConfig(rawConfig)

		warns, errs := r.Validate(config)
		if !reflect.DeepEqual(tc.ExpectedErrors, errs) {
			t.Fatalf("expected errors: %v, got: %v", tc.ExpectedErrors, errs)
		}
		if !reflect.DeepEqual(tc.ExpectedWarnings, warns) {
			t.Fatalf("expected warnings: %v, got: %v", tc.ExpectedWarnings, warns)
		}
	}
}

func TestResourceValidate_ComposeTestFunc(t *testing.T) {
	errorFooGreaterBar := func(g ResourceConfigGetter) (ws []string, es []error) {
		foo, fooKnown := g.Get("foo")
		bar, barKnown := g.Get("bar")

		// Cannot perform validation unless both foo and bar are known
		if !fooKnown || !barKnown {
			return
		}

		if foo.(int) > bar.(int) {
			es = append(es, fmt.Errorf("Foo cannot be greater than bar!"))
		}
		return
	}
	warnFooEqualBar := func(g ResourceConfigGetter) (ws []string, es []error) {
		foo, fooKnown := g.Get("foo")
		bar, barKnown := g.Get("bar")

		// Cannot perform validation unless both foo and bar are known
		if !fooKnown || !barKnown {
			return
		}

		if foo.(int) == bar.(int) {
			ws = append(ws, fmt.Sprintf("Foo is almost greater than bar!"))
		}
		return
	}
	r := &Resource{
		Schema: map[string]*Schema{
			"foo": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
			"bar": &Schema{
				Type:     TypeInt,
				Optional: true,
			},
		},
		ValidateFunc: ComposeResourceValidateFunc(
			errorFooGreaterBar,
			warnFooEqualBar,
		),
	}

	cases := map[string]struct {
		Config           map[string]interface{}
		ExpectedErrors   []error
		ExpectedWarnings []string
		Vars             map[string]ast.Variable
	}{
		"populates errors": {
			Config: map[string]interface{}{
				"foo": 5,
				"bar": 3,
			},
			ExpectedErrors: []error{errors.New("Foo cannot be greater than bar!")},
		},
		"populates warnings": {
			Config: map[string]interface{}{
				"foo": 3,
				"bar": 3,
			},
			ExpectedWarnings: []string{"Foo is almost greater than bar!"},
		},
		"can pass": {
			Config: map[string]interface{}{
				"foo": 3,
				"bar": 4,
			},
		},
		"validatefuncs can check for known/unknown": {
			Config: map[string]interface{}{
				"foo": "${var.notset}",
				"bar": 4,
			},
			Vars: map[string]ast.Variable{
				"var.notset": ast.Variable{
					Value: config.UnknownVariableValue,
					Type:  ast.TypeString,
				},
			},
		},
	}

	for _, tc := range cases {
		rawConfig, err := config.NewRawConfig(tc.Config)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		if err := rawConfig.Interpolate(tc.Vars); err != nil {
			t.Fatalf("err: %s", err)
		}
		config := terraform.NewResourceConfig(rawConfig)

		warns, errs := r.Validate(config)
		if !reflect.DeepEqual(tc.ExpectedErrors, errs) {
			t.Fatalf("expected errors: %v, got: %v", tc.ExpectedErrors, errs)
		}
		if !reflect.DeepEqual(tc.ExpectedWarnings, warns) {
			t.Fatalf("expected warnings: %v, got: %v", tc.ExpectedWarnings, warns)
		}
	}
}
