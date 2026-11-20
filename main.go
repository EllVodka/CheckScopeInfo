package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	mssql "github.com/microsoft/go-mssqldb"
	_ "github.com/microsoft/go-mssqldb/integratedauth/winsspi"
	_ "github.com/microsoft/go-mssqldb/namedpipe"
)

// Équivalent de .\SQLEXPRESS, via named pipe (le service SQL Browser est désactivé).
const connectionString = `Server=np:.;pipe=MSSQL$SQLEXPRESS\sql\query;Database=GDS;Integrated Security=True`

const outputFile = "scope_info.txt"

// Largeur max des colonnes *_knowledge (très longues), console et fichier.
const knowledgeMax = 40

type ScopeInfo struct {
	LocalID                   int64
	ScopeID                   mssql.UniqueIdentifier
	SyncScopeName             sql.NullString
	SyncKnowledge             []byte
	TombstoneCleanupKnowledge []byte
	Timestamp                 []byte
	ConfigID                  mssql.UniqueIdentifier
	RestoreCount              sql.NullInt64
	UserComment               sql.NullString
}

func main() {
	err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Erreur:", err)
	}
	fmt.Print("\nAppuyez sur un Entrée pour fermer...")
	_, err = bufio.NewReader(os.Stdin).ReadByte()
	if err != nil {
		os.Exit(1)
	}
}

func run() error {
	db, err := sql.Open("sqlserver", connectionString)
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT scope_local_id, scope_id, sync_scope_name, scope_sync_knowledge,
		scope_tombstone_cleanup_knowledge, scope_timestamp, scope_config_id, scope_restore_count,
		scope_user_comment FROM [GDS].[dbo].[scope_info]`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var list []ScopeInfo
	for rows.Next() {
		var s ScopeInfo
		if err := rows.Scan(&s.LocalID, &s.ScopeID, &s.SyncScopeName, &s.SyncKnowledge,
			&s.TombstoneCleanupKnowledge, &s.Timestamp, &s.ConfigID, &s.RestoreCount,
			&s.UserComment); err != nil {
			return err
		}
		list = append(list, s)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	// Console: valeurs tronquées ; fichier: valeurs complètes.
	print(os.Stdout, list, 40)
	print(f, list, 0)
	fmt.Printf("\nRésultat complet écrit dans %s\n", outputFile)
	return nil
}

func print(out io.Writer, list []ScopeInfo, max int) {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "scope_local_id\tscope_id\tsync_scope_name\tscope_sync_knowledge\tscope_tombstone_cleanup_knowledge\tscope_timestamp\tscope_config_id\tscope_restore_count\tscope_user_comment")
	for _, s := range list {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			s.LocalID, s.ScopeID.String(), cut(str(s.SyncScopeName), max),
			cut(hex(s.SyncKnowledge), knowledgeMax), cut(hex(s.TombstoneCleanupKnowledge), knowledgeMax),
			cut(hex(s.Timestamp), max), s.ConfigID.String(),
			num(s.RestoreCount), cut(str(s.UserComment), max))
	}
	w.Flush()
	fmt.Fprintf(out, "\n%d ligne(s)\n", len(list))
}

func hex(b []byte) string {
	if b == nil {
		return "NULL"
	}
	return fmt.Sprintf("0x%X", b)
}

func str(s sql.NullString) string {
	if !s.Valid {
		return "NULL"
	}
	return s.String
}

func num(n sql.NullInt64) string {
	if !n.Valid {
		return "NULL"
	}
	return fmt.Sprint(n.Int64)
}

func cut(s string, max int) string {
	if r := []rune(s); max > 3 && len(r) > max {
		return string(r[:max-3]) + "..."
	}
	return s
}
