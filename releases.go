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

type Releases struct {
	Releases []Release `json:"releases,omitempty"`

	Pagination Pagination `json:"pagination,omitempty"`
}
