package template_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"notification-service/internal/template"
)

func welcome() *template.Template {
	return &template.Template{
		ID:      "welcome",
		Name:    "Welcome",
		Subject: "Hi {{.Name}}!",
		Body:    "Welcome aboard, {{.Name}}.",
		Channel: "email",
	}
}

// --- RegisterTemplate ----------------------------------------------------

func TestEngine_RegisterTemplate_OK(t *testing.T) {
	e := template.NewEngine()
	require.NoError(t, e.RegisterTemplate(welcome()))
	assert.Len(t, e.ListTemplates(), 1)
}

func TestEngine_RegisterTemplate_NilReturnsError(t *testing.T) {
	e := template.NewEngine()
	assert.Error(t, e.RegisterTemplate(nil))
}

func TestEngine_RegisterTemplate_EmptyIDReturnsError(t *testing.T) {
	e := template.NewEngine()
	assert.Error(t, e.RegisterTemplate(&template.Template{Subject: "x", Body: "y"}))
}

func TestEngine_RegisterTemplate_InvalidSubjectSyntaxReturnsError(t *testing.T) {
	e := template.NewEngine()
	assert.Error(t, e.RegisterTemplate(&template.Template{
		ID: "bad", Subject: "{{.Unclosed", Body: "ok",
	}))
}

func TestEngine_RegisterTemplate_InvalidBodySyntaxReturnsError(t *testing.T) {
	e := template.NewEngine()
	assert.Error(t, e.RegisterTemplate(&template.Template{
		ID: "bad", Subject: "ok", Body: "{{.Unclosed",
	}))
}

// --- GetTemplate ---------------------------------------------------------

func TestEngine_GetTemplate_ExistingID(t *testing.T) {
	e := template.NewEngine()
	require.NoError(t, e.RegisterTemplate(welcome()))
	tmpl, err := e.GetTemplate("welcome")
	require.NoError(t, err)
	assert.Equal(t, "welcome", tmpl.ID)
}

func TestEngine_GetTemplate_UnknownIDReturnsError(t *testing.T) {
	e := template.NewEngine()
	_, err := e.GetTemplate("nope")
	assert.Error(t, err)
}

// --- RenderTemplate ------------------------------------------------------

func TestEngine_RenderTemplate_InterpolatesData(t *testing.T) {
	e := template.NewEngine()
	require.NoError(t, e.RegisterTemplate(welcome()))

	rendered, err := e.RenderTemplate("welcome", map[string]interface{}{"Name": "Ada"})
	require.NoError(t, err)
	assert.Equal(t, "Hi Ada!", rendered.Subject)
	assert.Equal(t, "Welcome aboard, Ada.", rendered.Body)
	assert.Equal(t, "email", rendered.Channel)
}

func TestEngine_RenderTemplate_UnknownIDReturnsError(t *testing.T) {
	e := template.NewEngine()
	_, err := e.RenderTemplate("nope", nil)
	assert.Error(t, err)
}

func TestEngine_Render_SatisfiesRendererInterface(t *testing.T) {
	e := template.NewEngine()
	require.NoError(t, e.RegisterTemplate(welcome()))

	subject, body, err := e.Render("welcome", map[string]interface{}{"Name": "Bob"})
	require.NoError(t, err)
	assert.Equal(t, "Hi Bob!", subject)
	assert.Equal(t, "Welcome aboard, Bob.", body)
}

// --- UpdateTemplate ------------------------------------------------------

func TestEngine_UpdateTemplate_ChangesSubjectAndBody(t *testing.T) {
	e := template.NewEngine()
	require.NoError(t, e.RegisterTemplate(welcome()))

	updated := &template.Template{
		ID: "welcome", Name: "Updated", Subject: "Hello {{.Name}}", Body: "Updated body for {{.Name}}", Channel: "slack",
	}
	require.NoError(t, e.UpdateTemplate("welcome", updated))

	rendered, err := e.RenderTemplate("welcome", map[string]interface{}{"Name": "Ada"})
	require.NoError(t, err)
	assert.Equal(t, "Hello Ada", rendered.Subject)
}

func TestEngine_UpdateTemplate_UnknownIDReturnsError(t *testing.T) {
	e := template.NewEngine()
	assert.Error(t, e.UpdateTemplate("nope", welcome()))
}

// --- DeleteTemplate ------------------------------------------------------

func TestEngine_DeleteTemplate_RemovesTemplate(t *testing.T) {
	e := template.NewEngine()
	require.NoError(t, e.RegisterTemplate(welcome()))
	require.NoError(t, e.DeleteTemplate("welcome"))
	assert.Empty(t, e.ListTemplates())
}

func TestEngine_DeleteTemplate_UnknownIDReturnsError(t *testing.T) {
	e := template.NewEngine()
	assert.Error(t, e.DeleteTemplate("nope"))
}

// --- ListTemplates -------------------------------------------------------

func TestEngine_ListTemplates_ReturnsAll(t *testing.T) {
	e := template.NewEngine()
	for _, id := range []string{"a", "b", "c"} {
		require.NoError(t, e.RegisterTemplate(&template.Template{
			ID: id, Subject: "s", Body: "b",
		}))
	}
	assert.Len(t, e.ListTemplates(), 3)
}
