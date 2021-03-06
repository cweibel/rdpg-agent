package rdpg

import (
	"fmt"
	"strings"

	"database/sql"

	_ "github.com/lib/pq"
	"github.com/wayneeseguin/rdpg-agent/log"
)

// TODO: This should only be run on one node...
func (r *RDPG) InitSchema() (err error) {
	// TODO: if 'rdpg' database DNE,
	// For each node connect to pgbdr and:
	//   CreatDatabase('rdpg','postgres')
	//   "ALTER USER postgres SUPERUSER CREATEDB CREATEROLE INHERIT"
	//   CreateReplicationGroup('rdpg')

	log.Trace(fmt.Sprintf("RDPG#initSchema() CONNECT > %s", rdpgURI))
	var name string
	r.OpenDB("rdpg")
	db := r.DB

	_, err = db.Exec(`SELECT bdr.bdr_node_join_wait_for_ready();`)
	if err != nil {
		log.Error(fmt.Sprintf("RDPG#initSchema() bdr.bdr_node_join_wait_for_ready ! %s", err))
	}

	keys := []string{
		"rdpg_extensions",
		"rdpg_schemas",
	}
	for _, key := range keys {
		log.Trace(fmt.Sprintf("RDPG#initSchema() SQL[%s]", key))
		_, err = db.Exec(SQL[key])
		if err != nil {
			log.Error(fmt.Sprintf("RDPG#initSchema() ! %s", err))
		}
	}

	keys = []string{
		"create_table_cfsb_services",
		"create_table_cfsb_plans",
		"create_table_cfsb_instances",
		"create_table_cfsb_bindings",
		"create_table_cfsb_credentials",
		"create_table_rdpg_watch_notifications",
	}
	for _, key := range keys {
		k := strings.Split(strings.Replace(strings.Replace(key, "create_table_", "", 1), "_", ".", 1), ".")
		sq := fmt.Sprintf(`SELECT table_name FROM information_schema.tables where table_schema='%s' AND table_name='%s';`, k[0], k[1])

		log.Trace(fmt.Sprintf("RDPG#initSchema() %s", sq))
		if err := db.QueryRow(sq).Scan(&name); err != nil {
			if err == sql.ErrNoRows {
				log.Trace(fmt.Sprintf("RDPG#initSchema() SQL[%s]", key))
				_, err = db.Exec(SQL[key])
				if err != nil {
					log.Error(fmt.Sprintf("RDPG#initSchema() ! %s", err))
				}
			} else {
				log.Error(fmt.Sprintf("rdpg.initSchema() ! %s", err))
				return err
			}
		}
	}

	// TODO: Move initial population of services out of rdpg-agent to Admin API.
	if err := db.QueryRow("SELECT name FROM cfsb.services WHERE name='rdpg' LIMIT 1;").Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			if _, err = db.Exec(SQL["insert_default_cfsb_services"]); err != nil {
				log.Error(fmt.Sprintf("rdpg.initSchema(insert_default_cfsb_services) %s", err))
				return err
			}
		} else {
			log.Error(fmt.Sprintf("rdpg.initSchema() ! %s", err))
			return err
		}
	}

	// TODO: Move initial population of services out of rdpg-agent to Admin API.
	if err = db.QueryRow("SELECT name FROM cfsb.plans WHERE name='shared' LIMIT 1;").Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			if _, err = db.Exec(SQL["insert_default_cfsb_plans"]); err != nil {
				log.Error(fmt.Sprintf("rdpg.initSchema(insert_default_cfsb_plans) %s", err))
				return err
			}
		} else {
			log.Error(fmt.Sprintf("rdpg.initSchema() ! %s", err))
			return err
		}
	}
	db.Close()

	for _, node := range r.Nodes() {
		node.Database = "postgres"
		db, err := node.Connect()
		if err != nil {
			log.Error(fmt.Sprintf("RDPG#DropUser(%s) %s ! %s", name, node.Host, err))
			return err
		}
		log.Trace(fmt.Sprintf("RDPG#initSchema() SQL[%s]", "postgres_schemas"))
		_, err = db.Exec(SQL["postgres_schemas"])
		if err != nil {
			log.Error(fmt.Sprintf("RDPG#initSchema() ! %s", err))
		}

		keys = []string{ // These are for the postgres database only
			"create_function_rdpg_disable_database",
		}
		for _, key := range keys {
			k := strings.Split(strings.Replace(strings.Replace(key, "create_function_", "", 1), "_", ".", 1), ".")
			sq := fmt.Sprintf(`SELECT routine_name FROM information_schema.routines WHERE routine_type='FUNCTION' AND routine_schema='%s' AND routine_name='%s';`, k[0], k[1])

			log.Trace(fmt.Sprintf("RDPG#initSchema() %s", sq))
			if err := db.QueryRow(sq).Scan(&name); err != nil {
				if err == sql.ErrNoRows {
					log.Trace(fmt.Sprintf("RDPG#initSchema() SQL[%s]", key))
					_, err = db.Exec(SQL[key])
					if err != nil {
						log.Error(fmt.Sprintf("RDPG#initSchema() %s", err))
					}
				} else {
					log.Error(fmt.Sprintf("rdpg.initSchema() %s", err))
					return err
				}
			}
		}
		db.Close()
	}
	return nil
}
