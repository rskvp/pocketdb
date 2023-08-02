package apis

import (
	"fmt"
	"log"
	"strings"

	"done/services/datastorage/daos"
	"done/services/datastorage/models"
	"done/services/datastorage/resolvers"
	"done/tools/rest"
	"done/tools/search"

	"github.com/ganigeorgiev/echo"
	"github.com/pocketbase/dbx"
)

const ContextRequestInfoKey = "requestInfo"

// Deprecated: Use RequestInfo instead.
func RequestData(c echo.Context) *models.RequestInfo {
	log.Println("RequestInfo(c) is depracated and will be removed in the future! You can replace it with RequestInfo(c).")
	return RequestInfo(c)
}

// RequestInfo exports cached common request data fields
// (query, body, logged auth state, etc.) from the provided context.
func RequestInfo(c echo.Context) *models.RequestInfo {
	// return cached to avoid copying the body multiple times
	if v := c.Get(ContextRequestInfoKey); v != nil {
		if data, ok := v.(*models.RequestInfo); ok {
			// refresh auth state
			data.AuthRecord, _ = c.Get(ContextAuthRecordKey).(*models.Record)
			data.Admin, _ = c.Get(ContextAdminKey).(*models.Admin)
			return data
		}
	}

	result := &models.RequestInfo{
		Method:  c.Request().Method,
		Query:   map[string]any{},
		Data:    map[string]any{},
		Headers: map[string]any{},
	}

	// extract the first value of all headers and normalizes the keys
	// ("X-Token" is converted to "x_token")
	for k, v := range c.Request().Header {
		if len(v) > 0 {
			result.Headers[strings.ToLower(strings.ReplaceAll(k, "-", "_"))] = v[0]
		}
	}

	result.AuthRecord, _ = c.Get(ContextAuthRecordKey).(*models.Record)
	result.Admin, _ = c.Get(ContextAdminKey).(*models.Admin)
	echo.BindQueryParams(c, &result.Query)
	rest.BindBody(c, &result.Data)

	c.Set(ContextRequestInfoKey, result)

	return result
}

// EnrichRecord parses the request context and enrich the provided record:
//   - expands relations (if defaultExpands and/or ?expand query param is set)
//   - ensures that the emails of the auth record and its expanded auth relations
//     are visibe only for the current logged admin, record owner or record with manage access
func EnrichRecord(c echo.Context, dao *daos.Dao, record *models.Record, defaultExpands ...string) error {
	return EnrichRecords(c, dao, []*models.Record{record}, defaultExpands...)
}

// EnrichRecords parses the request context and enriches the provided records:
//   - expands relations (if defaultExpands and/or ?expand query param is set)
//   - ensures that the emails of the auth records and their expanded auth relations
//     are visibe only for the current logged admin, record owner or record with manage access
func EnrichRecords(c echo.Context, dao *daos.Dao, records []*models.Record, defaultExpands ...string) error {
	requestInfo := RequestInfo(c)

	expands := defaultExpands
	if param := c.QueryParam(expandQueryParam); param != "" {
		expands = append(expands, strings.Split(param, ",")...)
	}
	if len(expands) == 0 {
		return nil // nothing to expand
	}

	errs := dao.ExpandRecords(records, expands, expandFetch(dao, requestInfo))
	if len(errs) > 0 {
		return fmt.Errorf("Failed to expand: %v", errs)
	}

	return nil
}

// expandFetch is the records fetch function that is used to expand related records.
func expandFetch(
	dao *daos.Dao,
	requestInfo *models.RequestInfo,
) daos.ExpandFetchFunc {
	return func(relCollection *models.Collection, relIds []string) ([]*models.Record, error) {
		records, err := dao.FindRecordsByIds(relCollection.Id, relIds, func(q *dbx.SelectQuery) error {
			if requestInfo.Admin != nil {
				return nil // admins can access everything
			}

			if relCollection.ViewRule == nil {
				return fmt.Errorf("Only admins can view collection %q records", relCollection.Name)
			}

			if *relCollection.ViewRule != "" {
				resolver := resolvers.NewRecordFieldResolver(dao, relCollection, requestInfo, true)
				expr, err := search.FilterData(*(relCollection.ViewRule)).BuildExpr(resolver)
				if err != nil {
					return err
				}
				resolver.UpdateQuery(q)
				q.AndWhere(expr)
			}

			return nil
		})

		return records, err
	}
}
