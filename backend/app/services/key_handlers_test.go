package services

import (
	"testing"
)

// ========== parseCommand 测试 ==========

func TestParseCommand_Basic(t *testing.T) {
	args := parseCommand("SET foo bar")
	if len(args) != 3 || args[0] != "SET" || args[1] != "foo" || args[2] != "bar" {
		t.Fatalf("unexpected: %v", args)
	}
}

func TestParseCommand_DoubleQuotes(t *testing.T) {
	args := parseCommand(`SET "hello world" "foo bar"`)
	if len(args) != 3 || args[1] != "hello world" || args[2] != "foo bar" {
		t.Fatalf("unexpected: %v", args)
	}
}

func TestParseCommand_SingleQuotes(t *testing.T) {
	args := parseCommand(`SET 'hello world' value`)
	if len(args) != 3 || args[1] != "hello world" {
		t.Fatalf("unexpected: %v", args)
	}
}

func TestParseCommand_EscapeChars(t *testing.T) {
	args := parseCommand(`SET key "val\"ue"`)
	if len(args) != 3 || args[2] != `val"ue` {
		t.Fatalf("unexpected: %v", args)
	}
}

func TestParseCommand_EscapeNewline(t *testing.T) {
	args := parseCommand(`SET key "line1\nline2"`)
	if len(args) != 3 || args[2] != "line1\nline2" {
		t.Fatalf("unexpected: %v", args)
	}
}

func TestParseCommand_EscapeTab(t *testing.T) {
	args := parseCommand(`SET key "col1\tcol2"`)
	if len(args) != 3 || args[2] != "col1\tcol2" {
		t.Fatalf("unexpected: %v", args)
	}
}

func TestParseCommand_Empty(t *testing.T) {
	args := parseCommand("")
	if len(args) != 0 {
		t.Fatalf("expected empty, got %v", args)
	}
}

func TestParseCommand_WhitespaceOnly(t *testing.T) {
	args := parseCommand("   \t  ")
	if len(args) != 0 {
		t.Fatalf("expected empty, got %v", args)
	}
}

func TestParseCommand_MultipleSpaces(t *testing.T) {
	args := parseCommand("GET   key1")
	if len(args) != 2 || args[0] != "GET" || args[1] != "key1" {
		t.Fatalf("unexpected: %v", args)
	}
}

func TestParseCommand_TabSeparated(t *testing.T) {
	args := parseCommand("SET\tkey\tvalue")
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %v", args)
	}
}

// ========== parseKeyspaceInfo 测试 ==========

func TestParseKeyspaceInfo_Normal(t *testing.T) {
	info := `# Keyspace
db0:keys=100,expires=10,avg_ttl=5000
db1:keys=50,expires=5,avg_ttl=3000
db3:keys=200,expires=0,avg_ttl=0
`
	result := parseKeyspaceInfo(info)
	if result[0] != 100 {
		t.Fatalf("db0 expected 100, got %d", result[0])
	}
	if result[1] != 50 {
		t.Fatalf("db1 expected 50, got %d", result[1])
	}
	if result[3] != 200 {
		t.Fatalf("db3 expected 200, got %d", result[3])
	}
	if _, ok := result[2]; ok {
		t.Fatal("db2 should not exist")
	}
}

func TestParseKeyspaceInfo_Empty(t *testing.T) {
	result := parseKeyspaceInfo("# Keyspace\n")
	if len(result) != 0 {
		t.Fatalf("expected empty map, got %v", result)
	}
}

// ========== parseInfoSections 测试 ==========

func TestParseInfoSections(t *testing.T) {
	info := `# Server
redis_version:7.0.0
redis_mode:standalone

# Clients
connected_clients:5

# Memory
used_memory:1024000
used_memory_human:1000.00K
`
	sections := parseInfoSections(info)

	if sections["Server"]["redis_version"] != "7.0.0" {
		t.Fatalf("unexpected redis_version: %s", sections["Server"]["redis_version"])
	}
	if sections["Server"]["redis_mode"] != "standalone" {
		t.Fatalf("unexpected redis_mode: %s", sections["Server"]["redis_mode"])
	}
	if sections["Clients"]["connected_clients"] != "5" {
		t.Fatalf("unexpected connected_clients: %s", sections["Clients"]["connected_clients"])
	}
	if sections["Memory"]["used_memory"] != "1024000" {
		t.Fatalf("unexpected used_memory: %s", sections["Memory"]["used_memory"])
	}
}

// ========== countTotalKeys 测试 ==========

func TestCountTotalKeys(t *testing.T) {
	keyspace := map[string]string{
		"db0": "keys=100,expires=10,avg_ttl=5000",
		"db1": "keys=50,expires=5,avg_ttl=3000",
	}
	total := countTotalKeys(keyspace)
	if total != 150 {
		t.Fatalf("expected 150, got %d", total)
	}
}

func TestCountTotalKeys_Empty(t *testing.T) {
	total := countTotalKeys(map[string]string{})
	if total != 0 {
		t.Fatalf("expected 0, got %d", total)
	}
}

// ========== formatResult 测试 ==========

func TestFormatResult_String(t *testing.T) {
	result := formatResult("hello")
	if result != `"hello"` {
		t.Fatalf("expected '\"hello\"', got %s", result)
	}
}

func TestFormatResult_Int64(t *testing.T) {
	result := formatResult(int64(42))
	if result != "(integer) 42" {
		t.Fatalf("expected '(integer) 42', got %s", result)
	}
}

func TestFormatResult_Nil(t *testing.T) {
	result := formatResult(nil)
	if result != "(nil)" {
		t.Fatalf("expected '(nil)', got %s", result)
	}
}

func TestFormatResult_EmptyArray(t *testing.T) {
	result := formatResult([]interface{}{})
	if result != "(empty array)" {
		t.Fatalf("expected '(empty array)', got %s", result)
	}
}

func TestFormatResult_Array(t *testing.T) {
	result := formatResult([]interface{}{"a", "b", "c"})
	expected := "1) \"a\"\n2) \"b\"\n3) \"c\""
	if result != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormatResult_NestedArray(t *testing.T) {
	result := formatResult([]interface{}{
		[]interface{}{"inner1", "inner2"},
		"outer",
	})
	if result == "" {
		t.Fatal("should not be empty")
	}
}
