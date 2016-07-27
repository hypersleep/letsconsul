package main

import(
	"fmt"
	"net"
	"time"
	"crypto"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/ecdsa"
	"errors"
	"strconv"
	"encoding/json"

	"github.com/xenolf/lego/acme"
	consul "github.com/hashicorp/consul/api"
)

type(
	DomainRecord struct {
		Domains    []string  `consul:"domains"`
		Email      string    `consul:"email"`
		Timestamp  time.Time `consul:"timestamp"`
		PrivateKey string    `consul:"private_key"`
		Fullchain  string    `consul:"fullchain"`
		key   *ecdsa.PrivateKey
		Reg   *acme.RegistrationResource
	}
)

func (domainRecord *DomainRecord) GetEmail() string { return domainRecord.Email }
func (domainRecord *DomainRecord) GetRegistration() *acme.RegistrationResource { return domainRecord.Reg }
func (domainRecord *DomainRecord) GetPrivateKey() crypto.PrivateKey { return domainRecord.key }

func (domainRecord *DomainRecord) write(client *consul.Client, domainRecordName string) error {
	kv := client.KV()
	consulService := "letsconsul"

	timestamp := domainRecord.Timestamp.Unix()
	timestampStr := strconv.Itoa(int(timestamp))

	p := &consul.KVPair {
		Key: consulService + "/domains/" + domainRecordName + "/timestamp",
		Value: []byte(timestampStr),
	}
	_, err := kv.Put(p, nil)
	if err != nil {
		return err
	}

	p = &consul.KVPair {
		Key: consulService + "/domains/" + domainRecordName + "/fullchain",
		Value: []byte(domainRecord.Fullchain),
	}
	_, err = kv.Put(p, nil)
	if err != nil {
		return err
	}

	p = &consul.KVPair {
		Key: consulService + "/domains/" + domainRecordName + "/private_key",
		Value: []byte(domainRecord.PrivateKey),
	}
	_, err = kv.Put(p, nil)
	if err != nil {
		return err
	}

	return nil
}

func (domainRecord *DomainRecord) renew(bind string) error {
	const letsEncryptURL = "https://acme-v01.api.letsencrypt.org/directory"

	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return err
	}

	domainRecord.key = key

	client, err := acme.NewClient(letsEncryptURL, domainRecord, acme.EC256)
	if err != nil {
		return err
	}

	reg, err := client.Register()
	if err != nil {
		return err
	}

	domainRecord.Reg = reg

	err = client.AgreeToTOS()
	if err != nil {
		return err
	}

	host, port, err := net.SplitHostPort(bind)
	if err != nil {
		return err
	}

	client.ExcludeChallenges([]acme.Challenge{acme.TLSSNI01})
	client.ExcludeChallenges([]acme.Challenge{acme.HTTP01})

	httpProvider := acme.NewHTTPProviderServer(host, port)

	err = client.SetChallengeProvider(acme.HTTP01, httpProvider)
	if err != nil {
		return err
	}

	acmeCert, errmap := client.ObtainCertificate(domainRecord.Domains, true, nil)
	if len(errmap) > 0 {
		err = fmt.Errorf("%v", errmap)
		return err
	}

	domainRecord.Fullchain = string(acmeCert.Certificate)
	domainRecord.PrivateKey = string(acmeCert.PrivateKey)

	domainRecord.Timestamp = time.Now()

	return nil
}

func kvFetch(kv *consul.KV, prefix string, domainRecordName string, key string) ([]byte, error) {
	kvPair, _, err := kv.Get(prefix + "/domains/" + domainRecordName + "/" + key, nil)
	if err != nil {
		return nil, err
	}

	if kvPair == nil {
		return nil, errors.New("Can't fetch '" + key + "' key from '" + domainRecordName + "' domain")
	}

	return kvPair.Value, nil
}

func (domainRecord *DomainRecord) get(kv *consul.KV, prefix string, domainRecordName string) error {
	v, err := kvFetch(kv, prefix, domainRecordName, "domain_list")
	if err != nil {
		return err
	}

	err = json.Unmarshal(v, &domainRecord.Domains)
	if err != nil {
		return err
	}

	v, err = kvFetch(kv, prefix, domainRecordName, "email")
	if err != nil {
		return err
	}

	domainRecord.Email = string(v)

	v, err = kvFetch(kv, prefix, domainRecordName, "timestamp")
	if err != nil {
		return err
	}

	i, err := strconv.ParseInt(string(v), 10, 64)
	if err != nil {
		return err
	}

	domainRecord.Timestamp = time.Unix(i, 0)

	return nil
}
