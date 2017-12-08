/*
 * Aha.io API
 *
 * Articles that matter on social publishing platform
 *
 * OpenAPI spec version: 1.0.0
 *
 * Generated by: https://github.com/swagger-api/swagger-codegen.git
 */

package aha

type InlineResponse200 struct {
	Features []InlineResponse200Features `json:"features,omitempty"`

	Pagination InlineResponse200Pagination `json:"pagination,omitempty"`
}
