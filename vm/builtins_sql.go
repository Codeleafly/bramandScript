package vm

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"bramandscript/object"
)

// SQLQueryPlan represents a parsed SQL SELECT query.
type SQLQueryPlan struct {
	SelectCols []SQLSelectItem
	FromTable  string
	FromAlias  string
	Joins      []SQLJoin
	WhereCond  *SQLExpr
	GroupBy    []string
	HavingCond *SQLExpr
	OrderBy    []SQLOrderItem
	Limit      int
	Offset     int
}

type SQLSelectItem struct {
	RawExpr    string
	Alias      string
	IsStar     bool
	IsAgg      bool
	AggFunc    string // COUNT, SUM, AVG, MIN, MAX
	AggArg     string // col name or "*"
	ColumnName string
}

type SQLJoin struct {
	Type       string // INNER, LEFT
	Table      string
	Alias      string
	LeftCol    string
	RightCol   string
	Comparison string
}

type SQLOrderItem struct {
	Column string
	Desc   bool
}

type SQLExpr struct {
	Op       string // =, !=, <, <=, >, >=, LIKE, IN, IS_NULL, IS_NOT_NULL, AND, OR
	Left     string
	Right    string
	InList   []string
	Children []*SQLExpr
}

// ParseSQL parses a SQL string into an SQLQueryPlan.
func ParseSQL(query string) (*SQLQueryPlan, error) {
	q := strings.TrimSpace(query)
	// Remove trailing semicolon
	q = strings.TrimSuffix(q, ";")

	plan := &SQLQueryPlan{
		Limit:  -1,
		Offset: 0,
	}

	tokens := tokenizeSQL(q)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty SQL query")
	}

	if strings.ToUpper(tokens[0]) != "SELECT" {
		return nil, fmt.Errorf("only SELECT queries are currently supported, got: %s", tokens[0])
	}

	idx := 1
	// 1. Parse SELECT columns
	var selectTokens []string
	for idx < len(tokens) && strings.ToUpper(tokens[idx]) != "FROM" {
		selectTokens = append(selectTokens, tokens[idx])
		idx++
	}

	if err := parseSelectClause(selectTokens, plan); err != nil {
		return nil, err
	}

	if idx >= len(tokens) || strings.ToUpper(tokens[idx]) != "FROM" {
		return nil, fmt.Errorf("missing FROM clause")
	}
	idx++ // skip FROM

	if idx >= len(tokens) {
		return nil, fmt.Errorf("missing table name after FROM")
	}
	plan.FromTable = tokens[idx]
	idx++

	if idx < len(tokens) && isTableAlias(tokens[idx]) {
		if strings.ToUpper(tokens[idx]) == "AS" {
			idx++
		}
		if idx < len(tokens) {
			plan.FromAlias = tokens[idx]
			idx++
		}
	}

	// 2. Parse JOINs, WHERE, GROUP BY, HAVING, ORDER BY, LIMIT
	for idx < len(tokens) {
		tokUpper := strings.ToUpper(tokens[idx])
		switch tokUpper {
		case "INNER", "LEFT", "JOIN":
			joinType := "INNER"
			if tokUpper == "LEFT" {
				joinType = "LEFT"
				idx++ // skip LEFT
				if idx < len(tokens) && strings.ToUpper(tokens[idx]) == "JOIN" {
					idx++
				}
			} else if tokUpper == "INNER" {
				joinType = "INNER"
				idx++ // skip INNER
				if idx < len(tokens) && strings.ToUpper(tokens[idx]) == "JOIN" {
					idx++
				}
			} else {
				idx++ // skip JOIN
			}

			if idx >= len(tokens) {
				return nil, fmt.Errorf("missing table name for JOIN")
			}
			joinTable := tokens[idx]
			idx++

			joinAlias := ""
			if idx < len(tokens) && isTableAlias(tokens[idx]) {
				if strings.ToUpper(tokens[idx]) == "AS" {
					idx++
				}
				if idx < len(tokens) {
					joinAlias = tokens[idx]
					idx++
				}
			}

			if idx >= len(tokens) || strings.ToUpper(tokens[idx]) != "ON" {
				return nil, fmt.Errorf("expected ON after JOIN table")
			}
			idx++ // skip ON

			if idx+2 >= len(tokens) {
				return nil, fmt.Errorf("malformed ON condition")
			}
			leftCol := tokens[idx]
			comp := tokens[idx+1]
			rightCol := tokens[idx+2]
			idx += 3

			plan.Joins = append(plan.Joins, SQLJoin{
				Type:       joinType,
				Table:      joinTable,
				Alias:      joinAlias,
				LeftCol:    leftCol,
				Comparison: comp,
				RightCol:   rightCol,
			})

		case "WHERE":
			idx++
			var whereTokens []string
			for idx < len(tokens) {
				tu := strings.ToUpper(tokens[idx])
				if tu == "GROUP" || tu == "HAVING" || tu == "ORDER" || tu == "LIMIT" {
					break
				}
				whereTokens = append(whereTokens, tokens[idx])
				idx++
			}
			cond, err := parseSQLExpr(whereTokens)
			if err != nil {
				return nil, err
			}
			plan.WhereCond = cond

		case "GROUP":
			idx++ // skip GROUP
			if idx < len(tokens) && strings.ToUpper(tokens[idx]) == "BY" {
				idx++ // skip BY
			}
			for idx < len(tokens) {
				tu := strings.ToUpper(tokens[idx])
				if tu == "HAVING" || tu == "ORDER" || tu == "LIMIT" {
					break
				}
				col := strings.Trim(tokens[idx], ",")
				if col != "" && col != "," {
					plan.GroupBy = append(plan.GroupBy, col)
				}
				idx++
			}

		case "HAVING":
			idx++
			var havingTokens []string
			for idx < len(tokens) {
				tu := strings.ToUpper(tokens[idx])
				if tu == "ORDER" || tu == "LIMIT" {
					break
				}
				havingTokens = append(havingTokens, tokens[idx])
				idx++
			}
			cond, err := parseSQLExpr(havingTokens)
			if err != nil {
				return nil, err
			}
			plan.HavingCond = cond

		case "ORDER":
			idx++ // skip ORDER
			if idx < len(tokens) && strings.ToUpper(tokens[idx]) == "BY" {
				idx++ // skip BY
			}
			for idx < len(tokens) {
				tu := strings.ToUpper(tokens[idx])
				if tu == "LIMIT" {
					break
				}
				col := strings.Trim(tokens[idx], ",")
				desc := false
				idx++
				if idx < len(tokens) {
					nxt := strings.ToUpper(strings.Trim(tokens[idx], ","))
					if nxt == "DESC" {
						desc = true
						idx++
					} else if nxt == "ASC" {
						idx++
					}
				}
				if col != "" && col != "," {
					plan.OrderBy = append(plan.OrderBy, SQLOrderItem{Column: col, Desc: desc})
				}
			}

		case "LIMIT":
			idx++
			if idx < len(tokens) {
				n, err := strconv.Atoi(tokens[idx])
				if err == nil {
					plan.Limit = n
				}
				idx++
			}
			if idx < len(tokens) && strings.ToUpper(tokens[idx]) == "OFFSET" {
				idx++
				if idx < len(tokens) {
					off, err := strconv.Atoi(tokens[idx])
					if err == nil {
						plan.Offset = off
					}
					idx++
				}
			}

		default:
			idx++
		}
	}

	return plan, nil
}

func tokenizeSQL(query string) []string {
	var tokens []string
	var cur strings.Builder
	inQuotes := false
	quoteChar := byte(0)

	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}

	for i := 0; i < len(query); i++ {
		ch := query[i]
		if inQuotes {
			if ch == quoteChar {
				inQuotes = false
			}
			cur.WriteByte(ch)
			continue
		}

		if ch == '\'' || ch == '"' || ch == '`' {
			flush()
			inQuotes = true
			quoteChar = ch
			cur.WriteByte(ch)
			continue
		}

		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			flush()
			continue
		}

		if ch == ',' || ch == '(' || ch == ')' {
			flush()
			tokens = append(tokens, string(ch))
			continue
		}

		if ch == '=' || ch == '!' || ch == '<' || ch == '>' {
			flush()
			if i+1 < len(query) {
				next := query[i+1]
				if (ch == '!' && next == '=') || (ch == '<' && (next == '=' || next == '>')) || (ch == '>' && next == '=') {
					tokens = append(tokens, string([]byte{ch, next}))
					i++
					continue
				}
			}
			tokens = append(tokens, string(ch))
			continue
		}

		cur.WriteByte(ch)
	}
	flush()
	return tokens
}

func parseSelectClause(tokens []string, plan *SQLQueryPlan) error {
	raw := strings.Join(tokens, " ")
	parts := strings.Split(raw, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		item := SQLSelectItem{RawExpr: part}
		pTokens := strings.Fields(part)
		if len(pTokens) >= 3 && strings.ToUpper(pTokens[len(pTokens)-2]) == "AS" {
			item.Alias = pTokens[len(pTokens)-1]
			pTokens = pTokens[:len(pTokens)-2]
			part = strings.Join(pTokens, " ")
		} else if len(pTokens) == 2 && !strings.Contains(pTokens[0], "(") {
			item.Alias = pTokens[1]
			part = pTokens[0]
		}

		partUpper := strings.ToUpper(part)
		partNorm := strings.ReplaceAll(partUpper, " ", "")
		if part == "*" {
			item.IsStar = true
		} else if strings.HasPrefix(partNorm, "COUNT(") && strings.HasSuffix(partNorm, ")") {
			item.IsAgg = true
			item.AggFunc = "COUNT"
			item.AggArg = partNorm[6 : len(partNorm)-1]
			if item.Alias == "" {
				item.Alias = part
			}
		} else if strings.HasPrefix(partNorm, "SUM(") && strings.HasSuffix(partNorm, ")") {
			item.IsAgg = true
			item.AggFunc = "SUM"
			item.AggArg = partNorm[4 : len(partNorm)-1]
			if item.Alias == "" {
				item.Alias = part
			}
		} else if strings.HasPrefix(partNorm, "AVG(") && strings.HasSuffix(partNorm, ")") {
			item.IsAgg = true
			item.AggFunc = "AVG"
			item.AggArg = partNorm[4 : len(partNorm)-1]
			if item.Alias == "" {
				item.Alias = part
			}
		} else if strings.HasPrefix(partNorm, "MIN(") && strings.HasSuffix(partNorm, ")") {
			item.IsAgg = true
			item.AggFunc = "MIN"
			item.AggArg = partNorm[4 : len(partNorm)-1]
			if item.Alias == "" {
				item.Alias = part
			}
		} else if strings.HasPrefix(partNorm, "MAX(") && strings.HasSuffix(partNorm, ")") {
			item.IsAgg = true
			item.AggFunc = "MAX"
			item.AggArg = partNorm[4 : len(partNorm)-1]
			if item.Alias == "" {
				item.Alias = part
			}
		} else {
			item.ColumnName = part
			if item.Alias == "" {
				item.Alias = part
			}
		}
		plan.SelectCols = append(plan.SelectCols, item)
	}
	return nil
}

func isTableAlias(tok string) bool {
	u := strings.ToUpper(tok)
	if u == "WHERE" || u == "INNER" || u == "LEFT" || u == "JOIN" || u == "GROUP" || u == "ORDER" || u == "LIMIT" || u == "ON" {
		return false
	}
	return true
}

func parseSQLExpr(tokens []string) (*SQLExpr, error) {
	if len(tokens) == 0 {
		return nil, nil
	}

	// Check for AND / OR top-level split
	andIdx := -1
	orIdx := -1
	for i, t := range tokens {
		u := strings.ToUpper(t)
		if u == "OR" && orIdx == -1 {
			orIdx = i
		} else if u == "AND" && andIdx == -1 {
			andIdx = i
		}
	}

	if orIdx != -1 {
		left, err := parseSQLExpr(tokens[:orIdx])
		if err != nil {
			return nil, err
		}
		right, err := parseSQLExpr(tokens[orIdx+1:])
		if err != nil {
			return nil, err
		}
		return &SQLExpr{
			Op:       "OR",
			Children: []*SQLExpr{left, right},
		}, nil
	}

	if andIdx != -1 {
		left, err := parseSQLExpr(tokens[:andIdx])
		if err != nil {
			return nil, err
		}
		right, err := parseSQLExpr(tokens[andIdx+1:])
		if err != nil {
			return nil, err
		}
		return &SQLExpr{
			Op:       "AND",
			Children: []*SQLExpr{left, right},
		}, nil
	}

	// Simple comparisons
	if len(tokens) >= 3 {
		left := tokens[0]
		op := tokens[1]
		right := tokens[2]

		opUpper := strings.ToUpper(op)
		if opUpper == "IS" && len(tokens) >= 3 {
			if strings.ToUpper(tokens[2]) == "NULL" {
				return &SQLExpr{Op: "IS_NULL", Left: left}, nil
			} else if len(tokens) >= 4 && strings.ToUpper(tokens[2]) == "NOT" && strings.ToUpper(tokens[3]) == "NULL" {
				return &SQLExpr{Op: "IS_NOT_NULL", Left: left}, nil
			}
		}

		if opUpper == "LIKE" {
			return &SQLExpr{
				Op:    "LIKE",
				Left:  left,
				Right: unquoteSQLString(right),
			}, nil
		}

		if opUpper == "IN" {
			var inList []string
			for _, item := range tokens[2:] {
				cleaned := strings.Trim(item, "(),'\"")
				if cleaned != "" {
					inList = append(inList, cleaned)
				}
			}
			return &SQLExpr{
				Op:     "IN",
				Left:   left,
				InList: inList,
			}, nil
		}

		return &SQLExpr{
			Op:    op,
			Left:  left,
			Right: unquoteSQLString(right),
		}, nil
	}

	return nil, fmt.Errorf("unsupported condition: %v", tokens)
}

func unquoteSQLString(s string) string {
	if (strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) ||
		(strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")) {
		return s[1 : len(s)-1]
	}
	return s
}

// ExecuteSQL runs an SQLQueryPlan over the dataset tables map.
func ExecuteSQL(plan *SQLQueryPlan, tables map[string][]map[string]interface{}) ([]map[string]interface{}, error) {
	fromData, ok := tables[plan.FromTable]
	if !ok {
		// Case-insensitive table search
		for k, v := range tables {
			if strings.EqualFold(k, plan.FromTable) {
				fromData = v
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil, fmt.Errorf("table '%s' not found", plan.FromTable)
	}

	// Prefix rows with table name or alias for disambiguation
	alias := plan.FromAlias
	if alias == "" {
		alias = plan.FromTable
	}

	var rows []map[string]interface{}
	for _, r := range fromData {
		rowCopy := make(map[string]interface{})
		for k, v := range r {
			rowCopy[k] = v
			rowCopy[alias+"."+k] = v
			rowCopy[plan.FromTable+"."+k] = v
		}
		rows = append(rows, rowCopy)
	}

	// 1. Execute JOINs
	for _, join := range plan.Joins {
		jData, found := tables[join.Table]
		if !found {
			for k, v := range tables {
				if strings.EqualFold(k, join.Table) {
					jData = v
					found = true
					break
				}
			}
		}
		if !found {
			return nil, fmt.Errorf("join table '%s' not found", join.Table)
		}

		jAlias := join.Alias
		if jAlias == "" {
			jAlias = join.Table
		}

		var joinedRows []map[string]interface{}
		for _, leftRow := range rows {
			matched := false
			leftVal := getRowValue(leftRow, join.LeftCol)

			for _, rightItem := range jData {
				rightVal := rightItem[join.RightCol]
				if rightVal == nil {
					// Check unqualified right column
					parts := strings.Split(join.RightCol, ".")
					rightVal = rightItem[parts[len(parts)-1]]
				}

				if compareSQLValues(leftVal, rightVal) == 0 {
					matched = true
					merged := make(map[string]interface{})
					for k, v := range leftRow {
						merged[k] = v
					}
					for k, v := range rightItem {
						merged[k] = v
						merged[jAlias+"."+k] = v
						merged[join.Table+"."+k] = v
					}
					joinedRows = append(joinedRows, merged)
				}
			}

			if !matched && join.Type == "LEFT" {
				merged := make(map[string]interface{})
				for k, v := range leftRow {
					merged[k] = v
				}
				joinedRows = append(joinedRows, merged)
			}
		}
		rows = joinedRows
	}

	// 2. Filter WHERE
	if plan.WhereCond != nil {
		var filtered []map[string]interface{}
		for _, r := range rows {
			if evalSQLExpr(plan.WhereCond, r) {
				filtered = append(filtered, r)
			}
		}
		rows = filtered
	}

	// 3. GROUP BY and Aggregations
	hasAggs := false
	for _, col := range plan.SelectCols {
		if col.IsAgg {
			hasAggs = true
			break
		}
	}

	if len(plan.GroupBy) > 0 || hasAggs {
		groups := make(map[string][]map[string]interface{})
		var groupOrder []string

		for _, r := range rows {
			key := ""
			for _, gCol := range plan.GroupBy {
				key += fmt.Sprintf("%v|", getRowValue(r, gCol))
			}
			if _, exists := groups[key]; !exists {
				groupOrder = append(groupOrder, key)
			}
			groups[key] = append(groups[key], r)
		}

		if len(rows) == 0 && hasAggs && len(plan.GroupBy) == 0 {
			// Zero rows for global aggregate e.g. COUNT(*) -> 0
			groups["global"] = []map[string]interface{}{}
			groupOrder = []string{"global"}
		}

		var aggRows []map[string]interface{}
		for _, gKey := range groupOrder {
			gRows := groups[gKey]
			aggRow := make(map[string]interface{})

			// Populate group columns
			if len(gRows) > 0 {
				first := gRows[0]
				for _, gCol := range plan.GroupBy {
					val := getRowValue(first, gCol)
					aggRow[gCol] = val
					parts := strings.Split(gCol, ".")
					aggRow[parts[len(parts)-1]] = val
				}
			}

			// Compute aggregates
			for _, sc := range plan.SelectCols {
				if sc.IsAgg {
					val := computeAggregate(sc.AggFunc, sc.AggArg, gRows)
					aggRow[sc.Alias] = val
					aggRow[sc.RawExpr] = val
				} else if len(gRows) > 0 {
					val := getRowValue(gRows[0], sc.ColumnName)
					aggRow[sc.Alias] = val
					aggRow[sc.ColumnName] = val
				}
			}

			// Filter HAVING
			if plan.HavingCond == nil || evalSQLExpr(plan.HavingCond, aggRow) {
				aggRows = append(aggRows, aggRow)
			}
		}
		rows = aggRows
	}

	// 4. ORDER BY
	if len(plan.OrderBy) > 0 {
		sort.SliceStable(rows, func(i, j int) bool {
			for _, ord := range plan.OrderBy {
				v1 := getRowValue(rows[i], ord.Column)
				v2 := getRowValue(rows[j], ord.Column)
				cmp := compareSQLValues(v1, v2)
				if cmp != 0 {
					if ord.Desc {
						return cmp > 0
					}
					return cmp < 0
				}
			}
			return false
		})
	}

	// 5. Final Projection
	var finalRows []map[string]interface{}
	for _, r := range rows {
		proj := make(map[string]interface{})
		for _, sc := range plan.SelectCols {
			if sc.IsStar {
				for k, v := range r {
					if !strings.Contains(k, ".") {
						proj[k] = v
					}
				}
			} else if sc.IsAgg {
				proj[sc.Alias] = r[sc.Alias]
			} else {
				proj[sc.Alias] = getRowValue(r, sc.ColumnName)
			}
		}
		finalRows = append(finalRows, proj)
	}

	// 6. OFFSET & LIMIT
	start := plan.Offset
	if start > len(finalRows) {
		start = len(finalRows)
	}
	end := len(finalRows)
	if plan.Limit >= 0 {
		end = start + plan.Limit
		if end > len(finalRows) {
			end = len(finalRows)
		}
	}

	return finalRows[start:end], nil
}

func getRowValue(row map[string]interface{}, colName string) interface{} {
	if val, ok := row[colName]; ok {
		return val
	}
	parts := strings.Split(colName, ".")
	lastPart := parts[len(parts)-1]
	if val, ok := row[lastPart]; ok {
		return val
	}
	for k, v := range row {
		if strings.EqualFold(k, colName) || strings.EqualFold(k, lastPart) {
			return v
		}
	}
	return nil
}

func evalSQLExpr(expr *SQLExpr, row map[string]interface{}) bool {
	if expr == nil {
		return true
	}

	switch expr.Op {
	case "AND":
		return evalSQLExpr(expr.Children[0], row) && evalSQLExpr(expr.Children[1], row)
	case "OR":
		return evalSQLExpr(expr.Children[0], row) || evalSQLExpr(expr.Children[1], row)
	case "IS_NULL":
		val := getRowValue(row, expr.Left)
		return val == nil
	case "IS_NOT_NULL":
		val := getRowValue(row, expr.Left)
		return val != nil
	case "LIKE":
		val := fmt.Sprintf("%v", getRowValue(row, expr.Left))
		pat := expr.Right
		return matchSQLLike(val, pat)
	case "IN":
		val := fmt.Sprintf("%v", getRowValue(row, expr.Left))
		for _, item := range expr.InList {
			if val == item {
				return true
			}
		}
		return false
	default:
		leftVal := getRowValue(row, expr.Left)
		var rightVal interface{} = expr.Right
		// If right matches another column name in row, treat as column comparison
		if colVal, exists := row[expr.Right]; exists {
			rightVal = colVal
		}

		cmp := compareSQLValues(leftVal, rightVal)
		switch expr.Op {
		case "=", "==":
			return cmp == 0
		case "!=", "<>":
			return cmp != 0
		case "<":
			return cmp < 0
		case "<=":
			return cmp <= 0
		case ">":
			return cmp > 0
		case ">=":
			return cmp >= 0
		}
	}
	return false
}

func matchSQLLike(str, pattern string) bool {
	pattern = strings.ToLower(pattern)
	str = strings.ToLower(str)

	if pattern == "%" {
		return true
	}
	if strings.HasPrefix(pattern, "%") && strings.HasSuffix(pattern, "%") {
		sub := pattern[1 : len(pattern)-1]
		return strings.Contains(str, sub)
	}
	if strings.HasPrefix(pattern, "%") {
		return strings.HasSuffix(str, pattern[1:])
	}
	if strings.HasSuffix(pattern, "%") {
		return strings.HasPrefix(str, pattern[:len(pattern)-1])
	}
	return str == pattern
}

func compareSQLValues(v1, v2 interface{}) int {
	if v1 == nil && v2 == nil {
		return 0
	}
	if v1 == nil {
		return -1
	}
	if v2 == nil {
		return 1
	}

	n1, ok1 := toFloat64(v1)
	n2, ok2 := toFloat64(v2)
	if ok1 && ok2 {
		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
		return 0
	}

	s1 := fmt.Sprintf("%v", v1)
	s2 := fmt.Sprintf("%v", v2)
	return strings.Compare(s1, s2)
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case float64:
		return val, true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return f, true
		}
	}
	return 0, false
}

func computeAggregate(aggFunc, aggArg string, rows []map[string]interface{}) interface{} {
	switch strings.ToUpper(aggFunc) {
	case "COUNT":
		if aggArg == "*" {
			return int64(len(rows))
		}
		count := int64(0)
		for _, r := range rows {
			if getRowValue(r, aggArg) != nil {
				count++
			}
		}
		return count

	case "SUM":
		sum := 0.0
		for _, r := range rows {
			val := getRowValue(r, aggArg)
			if f, ok := toFloat64(val); ok {
				sum += f
			}
		}
		return sum

	case "AVG":
		if len(rows) == 0 {
			return 0.0
		}
		sum := 0.0
		count := 0
		for _, r := range rows {
			val := getRowValue(r, aggArg)
			if f, ok := toFloat64(val); ok {
				sum += f
				count++
			}
		}
		if count == 0 {
			return 0.0
		}
		return sum / float64(count)

	case "MIN":
		minVal := math.MaxFloat64
		found := false
		for _, r := range rows {
			val := getRowValue(r, aggArg)
			if f, ok := toFloat64(val); ok {
				if f < minVal {
					minVal = f
					found = true
				}
			}
		}
		if !found {
			return nil
		}
		return minVal

	case "MAX":
		maxVal := -math.MaxFloat64
		found := false
		for _, r := range rows {
			val := getRowValue(r, aggArg)
			if f, ok := toFloat64(val); ok {
				if f > maxVal {
					maxVal = f
					found = true
				}
			}
		}
		if !found {
			return nil
		}
		return maxVal
	}
	return nil
}

// ConvertBramandTables converts a BramandScript Hash of tables (hashes of arrays) to Go map.
func ConvertBramandTables(tablesObj object.Object) (map[string][]map[string]interface{}, error) {
	tablesHash, ok := tablesObj.(*object.Hash)
	if !ok {
		return nil, fmt.Errorf("tables must be a Hash of table names to arrays of records")
	}

	result := make(map[string][]map[string]interface{})
	for _, pair := range tablesHash.Pairs {
		tableName := pair.Key.Inspect()
		if sObj, ok := pair.Key.(*object.String); ok {
			tableName = sObj.Value
		}

		var rowList []map[string]interface{}
		switch val := pair.Value.(type) {
		case *object.Array:
			for _, el := range val.Elements {
				if rowHash, ok := el.(*object.Hash); ok {
					row := make(map[string]interface{})
					for _, rp := range rowHash.Pairs {
						k := rp.Key.Inspect()
						if s, ok := rp.Key.(*object.String); ok {
							k = s.Value
						}
						row[k] = unwrapBramandObject(rp.Value)
					}
					rowList = append(rowList, row)
				}
			}
		case *object.Hash:
			// Could be a DataFrame with "rows"
			rowsKey := (&object.String{Value: "rows"}).HashKey()
			if rPair, exists := val.Pairs[rowsKey]; exists {
				if rArr, ok := rPair.Value.(*object.Array); ok {
					for _, el := range rArr.Elements {
						if rowHash, ok := el.(*object.Hash); ok {
							row := make(map[string]interface{})
							for _, rp := range rowHash.Pairs {
								k := rp.Key.Inspect()
								if s, ok := rp.Key.(*object.String); ok {
									k = s.Value
								}
								row[k] = unwrapBramandObject(rp.Value)
							}
							rowList = append(rowList, row)
						}
					}
				}
			}
		}
		result[tableName] = rowList
	}
	return result, nil
}

func unwrapBramandObject(obj object.Object) interface{} {
	if obj == nil || obj == Null {
		return nil
	}
	switch v := obj.(type) {
	case *object.Null:
		return nil
	case *object.Integer:
		return v.Value
	case *object.Float:
		return v.Value
	case *object.String:
		return v.Value
	case *object.Boolean:
		return v.Value
	default:
		return v.Inspect()
	}
}

func wrapResultToBramandArray(records []map[string]interface{}) *object.Array {
	var elements []object.Object
	for _, rec := range records {
		h := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		for k, v := range rec {
			keyObj := &object.String{Value: k}
			var valObj object.Object
			if v == nil {
				valObj = Null
			} else {
				switch val := v.(type) {
				case int:
					valObj = &object.Integer{Value: int64(val)}
				case int64:
					valObj = &object.Integer{Value: val}
				case float64:
					valObj = &object.Float{Value: val}
				case string:
					valObj = &object.String{Value: val}
				case bool:
					if val {
						valObj = True
					} else {
						valObj = False
					}
				default:
					valObj = &object.String{Value: fmt.Sprintf("%v", val)}
				}
			}
			h.Pairs[keyObj.HashKey()] = object.HashPair{Key: keyObj, Value: valObj}
		}
		elements = append(elements, h)
	}
	return &object.Array{Elements: elements}
}

func init() {
	RegisterBuiltin("sql_query", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return &object.Error{Message: "sql_query requires 2 arguments: (queryStr, tables)"}
			}

			queryStr := args[0].Inspect()
			if s, ok := args[0].(*object.String); ok {
				queryStr = s.Value
			}

			tables, err := ConvertBramandTables(args[1])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}

			plan, err := ParseSQL(queryStr)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("SQL parse error: %s", err)}
			}

			res, err := ExecuteSQL(plan, tables)
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("SQL execution error: %s", err)}
			}

			return wrapResultToBramandArray(res)
		},
	})
}
