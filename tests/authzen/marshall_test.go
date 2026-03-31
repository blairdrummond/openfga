package authzen_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	authzenv1 "github.com/openfga/api/proto/authzen/v1"
	openfgav1 "github.com/openfga/api/proto/openfga/v1"
)

func TestMarshal(t *testing.T) {
	tests := []struct {
		description string
		inputJSON   string
		expected    *authzenv1.EvaluationRequest
	}{
		{
			description: "basic_evaluation_request",
			inputJSON: `{
				"subject": {"type": "user", "id": "alice"},
				"resource": {"type": "document", "id": "doc1"},
				"action": {"name": "reader"}
			}`,
			expected: &authzenv1.EvaluationRequest{
				Subject:  &authzenv1.Subject{Type: "user", Id: "alice"},
				Resource: &authzenv1.Resource{Type: "document", Id: "doc1"},
				Action:   &authzenv1.Action{Name: "reader"},
			},
		},
		{
			description: "evaluation_request_with_context",
			inputJSON: `{
				"subject": {"type": "user", "id": "bob"},
				"resource": {"type": "file", "id": "file1"},
				"action": {"name": "write"},
				"context": {
					"consistency": "HIGHER_CONSISTENCY",
					"tuples": {
						"tuple_keys": [
							{
								"user": "user:charlie",
								"relation": "owner",
								"object": "file:file2"
							}
						]
					},
					"data": {
						"custom_field": "custom_value"
					}
				}
			}`,
			expected: &authzenv1.EvaluationRequest{
				Subject:  &authzenv1.Subject{Type: "user", Id: "bob"},
				Resource: &authzenv1.Resource{Type: "file", Id: "file1"},
				Action:   &authzenv1.Action{Name: "write"},
				Context: &authzenv1.Context{
					Consistency: openfgav1.ConsistencyPreference_HIGHER_CONSISTENCY.Enum(),
					Tuples: &openfgav1.ContextualTupleKeys{
						TupleKeys: []*openfgav1.TupleKey{
							{
								User:     "user:charlie",
								Relation: "owner",
								Object:   "file:file2",
							},
						},
					},
					Data: mustNewStruct(t, map[string]interface{}{
						"custom_field": "custom_value",
					}),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			var actual authzenv1.EvaluationRequest
			err := protojson.Unmarshal([]byte(tt.inputJSON), &actual)
			require.NoError(t, err)

			require.Equal(t, tt.expected.GetSubject().GetType(), actual.GetSubject().GetType())
			require.Equal(t, tt.expected.GetSubject().GetId(), actual.GetSubject().GetId())
			require.Equal(t, tt.expected.GetResource().GetType(), actual.GetResource().GetType())
			require.Equal(t, tt.expected.GetResource().GetId(), actual.GetResource().GetId())
			require.Equal(t, tt.expected.GetAction().GetName(), actual.GetAction().GetName())

			if tt.expected.GetContext() != nil {
				require.NotNil(t, actual.GetContext())
				require.Equal(t, tt.expected.GetContext().GetConsistency(), actual.GetContext().GetConsistency())

				if tt.expected.GetContext().GetTuples() != nil {
					require.NotNil(t, actual.GetContext().GetTuples())
					require.Len(t, actual.GetContext().GetTuples().GetTupleKeys(), len(tt.expected.GetContext().GetTuples().GetTupleKeys()))
					for i, expectedTuple := range tt.expected.GetContext().GetTuples().GetTupleKeys() {
						actualTuple := actual.GetContext().GetTuples().GetTupleKeys()[i]
						require.Equal(t, expectedTuple.GetUser(), actualTuple.GetUser())
						require.Equal(t, expectedTuple.GetRelation(), actualTuple.GetRelation())
						require.Equal(t, expectedTuple.GetObject(), actualTuple.GetObject())
					}
				}

				if tt.expected.GetContext().GetData() != nil {
					require.NotNil(t, actual.GetContext().GetData())
					require.Equal(t, tt.expected.GetContext().GetData().AsMap(), actual.GetContext().GetData().AsMap())
				}
			}
		})
	}
}

func TestRoundtrip(t *testing.T) {
	t.Run("struct_to_json_to_struct_identity", func(t *testing.T) {
		// Create original struct
		original := &authzenv1.EvaluationRequest{
			Subject:  &authzenv1.Subject{Type: "user", Id: "alice"},
			Resource: &authzenv1.Resource{Type: "document", Id: "doc1"},
			Action:   &authzenv1.Action{Name: "reader"},
			Context: &authzenv1.Context{
				Consistency: openfgav1.ConsistencyPreference_MINIMIZE_LATENCY.Enum(),
				Tuples: &openfgav1.ContextualTupleKeys{
					TupleKeys: []*openfgav1.TupleKey{
						{
							User:     "user:bob",
							Relation: "owner",
							Object:   "document:doc2",
						},
					},
				},
				Data: mustNewStruct(t, map[string]interface{}{
					"field1": "value1",
					"field2": 42.0,
				}),
			},
		}

		// Marshal to JSON
		jsonBytes, err := protojson.Marshal(original)
		require.NoError(t, err)

		// Unmarshal back to struct
		var roundtripped authzenv1.EvaluationRequest
		err = protojson.Unmarshal(jsonBytes, &roundtripped)
		require.NoError(t, err)

		// Verify identity using proto.Equal
		require.True(t, proto.Equal(original, &roundtripped), "structs should be equal after roundtrip")
	})

	t.Run("json_to_struct_to_json_identity", func(t *testing.T) {
		// Original JSON
		originalJSON := `{
			"subject": {"type": "user", "id": "charlie"},
			"resource": {"type": "file", "id": "file3"},
			"action": {"name": "delete"},
			"context": {
				"consistency": "HIGHER_CONSISTENCY",
				"tuples": {
					"tuple_keys": [
						{
							"user": "user:dave",
							"relation": "editor",
							"object": "file:file4"
						}
					]
				},
				"data": {
					"key1": "value1",
					"key2": 123
				}
			}
		}`

		// Unmarshal to struct
		var parsedStruct authzenv1.EvaluationRequest
		err := protojson.Unmarshal([]byte(originalJSON), &parsedStruct)
		require.NoError(t, err)

		// Verify the struct was parsed correctly
		require.Equal(t, "charlie", parsedStruct.GetSubject().GetId())
		require.Equal(t, "file3", parsedStruct.GetResource().GetId())
		require.Equal(t, "delete", parsedStruct.GetAction().GetName())
		require.Equal(t, openfgav1.ConsistencyPreference_HIGHER_CONSISTENCY, parsedStruct.GetContext().GetConsistency())
		require.Len(t, parsedStruct.GetContext().GetTuples().GetTupleKeys(), 1)
		require.Equal(t, "user:dave", parsedStruct.GetContext().GetTuples().GetTupleKeys()[0].GetUser())

		// Marshal back to JSON
		roundtrippedJSON, err := protojson.Marshal(&parsedStruct)
		require.NoError(t, err)

		// Unmarshal the roundtripped JSON back to struct to verify it's still valid
		var finalStruct authzenv1.EvaluationRequest
		err = protojson.Unmarshal(roundtrippedJSON, &finalStruct)
		require.NoError(t, err)

		// Verify final struct matches the parsed struct using proto.Equal
		require.True(t, proto.Equal(&parsedStruct, &finalStruct), "structs should be equal after JSON roundtrip")
	})
}

func mustNewStruct(t *testing.T, m map[string]interface{}) *structpb.Struct {
	t.Helper()
	data, err := json.Marshal(m)
	require.NoError(t, err)
	var s structpb.Struct
	err = protojson.Unmarshal(data, &s)
	require.NoError(t, err)
	return &s
}
