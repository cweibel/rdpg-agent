package cfsb

import (
	"fmt"
	"strings"

	"github.com/wayneeseguin/rdpg-agent/log"
	"github.com/wayneeseguin/rdpg-agent/rdpg"
)

type Credentials struct {
	URI      string `json:"uri"`
	DSN      string `json:"dsn"`
	JDBCURI  string `json:"jdbc_uri"`
	Host     string `db:"host" json:"host"`
	Port     string `db:"port" json:"port"`
	UserName string `db:"username" json:"username"`
	Password string `db:"password" json:"password"`
	Database string `db:"database" json:"database"`
}

type Binding struct {
	BindingId  string      `json:"binding_id"`
	InstanceId string      `json:"instance_id"`
	Creds      Credentials `json:"credentials"`
}

func CreateBinding(instanceId, bindingId string) (binding *Binding, err error) {
	log.Trace(fmt.Sprintf("cfsb.CreateBinding(%s,%s)", instanceId, bindingId))
	instance, err := FindInstance(instanceId)
	if err != nil {
		return
	}
	binding = &Binding{BindingId: bindingId, InstanceId: instanceId}

	dns := instance.ExternalDNS()
	s := strings.Split(dns, ":")

	binding.Creds = Credentials{
		URI:      instance.URI(),
		DSN:      instance.DSN(),
		JDBCURI:  "jdbc:" + instance.URI(),
		Host:     s[0],
		Port:     s[1],
		UserName: instance.User,
		Password: instance.Pass,
		Database: instance.Database,
	}

	r := rdpg.New()
	r.OpenDB()

	sq := `INSERT INTO cfsb.bindings (instance_id,binding_id) VALUES ($1,$2);`
	_, err = r.DB.Query(sq, binding.InstanceId, binding.BindingId)
	if err != nil {
		log.Error(fmt.Sprintf(`cfsb.CreateBinding() %s`, err))
	}

	sq = `INSERT INTO cfsb.credentials (instance_id,binding_id,host,port,uname,pass,dbname) VALUES ($1,$2,$3,$4,$5,$6,$7);`
	_, err = r.DB.Query(sq, binding.InstanceId, binding.BindingId, binding.Creds.Host, binding.Creds.Port, binding.Creds.UserName, binding.Creds.Password, binding.Creds.Database)
	if err != nil {
		log.Error(fmt.Sprintf(`cfsb.CreateBinding() %s`, err))
	}

	return
}

func RemoveBinding(bindingId string) (binding *Binding, err error) {
	log.Trace(fmt.Sprintf("cfsb.RemoveBinding(%s)", bindingId))
	binding, err = FindBinding(bindingId)
	if err != nil {
		return
	}
	r := rdpg.New()
	r.OpenDB()

	sq := `UPDATE cfsb.bindings SET ineffective_at=CURRENT_TIMESTAMP WHERE binding_id = $1;`
	_, err = r.DB.Query(sq, binding.BindingId)
	if err != nil {
		log.Error(fmt.Sprintf(`cfsb.CreateBinding() %s`, err))
	}
	return
}

func FindBinding(bindingId string) (binding *Binding, err error) {
	r := rdpg.New()
	r.OpenDB()
	b := Binding{}
	sq := `SELECT instance_id, binding_id FROM cfsb.bindings WHERE binding_id=lower($1) LIMIT 1;`
	err = r.DB.Get(&b, sq, bindingId)
	if err != nil {
		// TODO: Change messaging if err is sql.NoRows then say couldn't find binding with bindingId
		log.Error(fmt.Sprintf("cfsb.FindBinding(%s) %s", bindingId, err))
	}
	r.DB.Close()
	binding = &b
	return
}
