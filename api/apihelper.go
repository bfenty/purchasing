package api

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Field struct {
	Column string
	Alias  string
}

func buildQuery(table string, selectedFields []Field, filterConditions map[string]string, queryType ...string) ([]interface{}, strings.Builder, string) {
	var queryArgs []interface{}
	var queryBuilder strings.Builder
	qType := "SELECT"
	if len(queryType) > 0 {
		qType = queryType[0]
	}

	switch qType {
	case "SELECT":
		queryBuilder.WriteString("SELECT ")
		fields := []string{}
		for _, field := range selectedFields {
			fields = append(fields, fmt.Sprintf("%s AS %s", field.Column, field.Alias))
		}
		queryBuilder.WriteString(strings.Join(fields, ", "))
		queryBuilder.WriteString(" FROM ")
		queryBuilder.WriteString(table)
		queryBuilder.WriteString(" WHERE 1")

		for column, value := range filterConditions {
			if value != "" {
				queryArgs = append(queryArgs, value)
				queryBuilder.WriteString(fmt.Sprintf(" AND %s = ?", column))
			}
		}

		countQuery := strings.Replace(queryBuilder.String(), "SELECT "+strings.Join(fields, ", "), "SELECT COUNT(*)", 1)
		return queryArgs, queryBuilder, countQuery

	case "REPLACE":
		queryBuilder.WriteString(fmt.Sprintf("REPLACE INTO %s (", table))
		columns := []string{}
		placeholders := []string{}
		for _, field := range selectedFields {
			columns = append(columns, field.Column)
			placeholders = append(placeholders, "?")
		}
		queryBuilder.WriteString(strings.Join(columns, ", "))
		queryBuilder.WriteString(") VALUES (")
		queryBuilder.WriteString(strings.Join(placeholders, ", "))
		queryBuilder.WriteString(")")

		for _, field := range selectedFields {
			if val, exists := filterConditions[field.Column]; exists {
				queryArgs = append(queryArgs, val)
			} else {
				queryArgs = append(queryArgs, nil)
			}
		}
		return queryArgs, queryBuilder, ""

	default:
		log.Error("Unsupported query type: ", qType)
	}

	return nil, queryBuilder, ""
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func derefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func derefInt(i *int) int {
	if i != nil {
		return *i
	}
	return 0
}

func derefFloat(f *float64) float64 {
	if f != nil {
		return *f
	}
	return 0.0
}

func derefBool(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}
