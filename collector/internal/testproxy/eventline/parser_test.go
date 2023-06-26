package eventline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("rejects events that don't start with @Event", func(t *testing.T) {
		_, err := Parse("@")
		require.ErrorContains(t, err, "parse error near")
	})

	t.Run("rejects events that don't have a valid millisecond start time", func(t *testing.T) {
		_, err := Parse("@Event 123abc")
		require.ErrorContains(t, err, "parse error near")
	})

	t.Run("rejects events that don't have a valid millisecond end time", func(t *testing.T) {
		_, err := Parse("@Event 123 456def")
		require.ErrorContains(t, err, "parse error near")
	})

	t.Run("rejects events with bad names", func(t *testing.T) {
		_, err := Parse("@Event 123 456 %name")
		require.ErrorContains(t, err, "parse error near")
	})

	t.Run("rejects events with invalid annotation key", func(t *testing.T) {
		_, err := Parse("@Event 123 456 name %annotation")
		require.ErrorContains(t, err, "parse error near")
	})

	t.Run("rejects events with invalid annotation value", func(t *testing.T) {
		_, err := Parse("@Event 123 456 name annotation=value")
		require.ErrorContains(t, err, "parse error near")
	})

	t.Run("parses events successfully", func(t *testing.T) {
		_, err := Parse("@Event 123 456 name annotation=\"value\"")
		require.NoError(t, err)
	})

	t.Run("parses out event name", func(t *testing.T) {
		event, _ := Parse("@Event 123 456 name annotation=\"value\"")
		require.Equal(t, "name", event.Name)
	})

	t.Run("parses out event timestamps", func(t *testing.T) {
		event, _ := Parse("@Event 123 456 name annotation=\"value\"")
		require.Equal(t, "123", event.Start, "start millis")
		require.Equal(t, "456", event.End, "end millis")
	})

	t.Run("parses out an annotation", func(t *testing.T) {
		event, _ := Parse("@Event 123 456 name annotation=\"value\"")
		require.Equal(t, "value", event.Annotations["annotation"])
	})

	t.Run("parses multiple annotations", func(t *testing.T) {
		event, _ := Parse("@Event 123 456 name someannotation=\"somevalue\" anotherannotation=\"anothervalue\"")
		require.Equal(t, "somevalue", event.Annotations["someannotation"])
		require.Equal(t, "anothervalue", event.Annotations["anotherannotation"])
	})

	t.Run("annotation values can have any character unescaped (except quotes)", func(t *testing.T) {
		event, _ := Parse("@Event 123 456 name someannotation=\"!@#$%^&*()abcd1231241234_+=:<>,.?/\"")
		require.Equal(t, "!@#$%^&*()abcd1231241234_+=:<>,.?/", event.Annotations["someannotation"])
	})

	t.Run("parses escaped quotes in annotations properly", func(t *testing.T) {
		event, _ := Parse(`@Event 123 456 name someannotation="foo: \"bar\""`)
		require.Equal(t, `foo: "bar"`, event.Annotations["someannotation"])
	})

	t.Run("rejects annotations which do not end with a quote", func(t *testing.T) {
		_, err := Parse(`@Event 123 456 name someannotation="foo`)
		require.ErrorContains(t, err, "parse error near")
	})

	t.Run("rejects annotations which do not end with a quote", func(t *testing.T) {
		_, err := Parse(`@Event 123 456 name someannotation="fooannother="bar"`)
		require.ErrorContains(t, err, "parse error near")
	})

	t.Run("parses tags separately", func(t *testing.T) {
		event, err := Parse(`@Event 123 456 name someannotation="foo" tag="\"tagKey\": \"tagValue\""`)

		require.NoError(t, err)

		require.Empty(t, event.Annotations["tag"])
		require.Equal(t, "tagValue", event.Tags["tagKey"])
	})

	t.Run("parses complex tag values", func(t *testing.T) {
		event, err := Parse(`@Event 123 456 name someannotation="foo" tag="\"tagKey\": \"!@#$%^&*()abcd1231241234_+=:<>,.?/\""`)

		require.NoError(t, err)

		require.Equal(t, "!@#$%^&*()abcd1231241234_+=:<>,.?/", event.Tags["tagKey"])
	})

	t.Run("parses complex tag values", func(t *testing.T) {
		_, err := Parse(`@Event 123 456 name someannotation="foo" tag="\"tagKey\": \""\""`)

		require.ErrorContains(t, err, "parse error near")
	})
}
