package bench

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	_ "github.com/go-kivik/couchdb/v4"
	"github.com/go-kivik/kivik/v4"
	log "github.com/sirupsen/logrus"
)

/*
 * Misc CouchDB related tooling, taken from sphincter
 */

type View struct {
	Map    string `json:"map,omitempty"`
	Reduce string `json:"reduce,omitempty"`
}

type DesignDoc struct {
	ID          string          `json:"_id"`
	Rev         string          `json:"_rev,omitempty"`
	Version     int             `json:"version"`
	Description string          `json:"-"`
	Views       map[string]View `json:"views"`
}

func Check(db *kivik.DB, AllDesignDocs []*DesignDoc) {
	for _, dd := range AllDesignDocs {
		docId := dd.ID
		log.Infof("Checking Design Document %s: %s", docId, dd.Description)
		doc := &DesignDoc{}
		found, err := GetOr404(db, docId, doc)
		if err != nil {
			panic(err)
		}
		if !found {
			log.WithField("id", docId).Warnf("DesignDocument does not exist, adding")
			_, err = db.Put(context.TODO(), docId, dd)
			if err != nil {
				panic(err)
			}
			continue
		}
		if doc.Version < dd.Version {
			log.WithFields(log.Fields{
				"id":          docId,
				"old_version": doc.Version,
				"new_version": dd.Version,
			}).Warnf("DesignDocument is out of date, updating")
			dd.Rev = doc.Rev
			_, err = db.Put(context.TODO(), docId, dd)
			if err != nil {
				if kivik.StatusCode(err) == http.StatusConflict {
					log.WithField("id", "docId").Warning("Ignoring conflict error while updating. Probably multiple processes starting/updating at once.")
				} else {
					panic(err)
				}
			}
			continue
		}
		log.Infof("Design Document %s: %s is up to date", docId, dd.Description)
	}
}

type WrappedDoc struct {
	ID       string      `json:"_id"`
	Rev      string      `json:"_rev,omitempty"`
	Type     string      `json:"type"`
	Document interface{} `json:"document"`
}

// WrapDoc takes any model and embeds it into something we can store in CouchDB
func WrapDoc(docId, rev, dtype string, doc interface{}) *WrappedDoc {
	d := &WrappedDoc{
		ID:       docId,
		Rev:      rev,
		Type:     dtype,
		Document: doc,
	}
	return d
}

// MakeDoc builds something that CouchDB can deserialize into, which is why we don't need a type or rev
func MakeDoc(doc interface{}) *WrappedDoc {
	return &WrappedDoc{
		Document: doc,
	}
}

// GetOr404 fetches the document or returns false if not found
func GetOr404(db *kivik.DB, docId string, doc interface{}) (bool, error) {
	row := db.Get(context.TODO(), docId)
	if row.Err != nil {
		switch kivik.StatusCode(row.Err) {
		case http.StatusNotFound:
			return false, nil
		}
		return false, row.Err
	}

	if err := row.ScanDoc(&doc); err != nil {
		return false, err
	}
	return true, nil
}

// PUTs the document with the given docId and, if fails because of conflict, fetches the
// latest rev and retries. Returns the new rev or error
func RevPut(db *kivik.DB, docId string, doc *WrappedDoc) (string, error) {
	rev, err := db.Put(context.TODO(), docId, doc)
	if err != nil && kivik.StatusCode(err) == http.StatusConflict {
		if _, rev, err = db.GetMeta(context.TODO(), docId); err != nil {
			return "", err
		}
		doc.Rev = rev
		if rev, err = db.Put(context.TODO(), docId, doc); err != nil {
			log.WithFields(log.Fields{
				"_id": docId,
				"err": err,
			}).Warn("Failed to save doc after 2 attempts")
			return "", err
		}
	}
	return rev, err // err may or may not be nil!
}

type CouchDB struct {
	client *kivik.Client
	db     *kivik.DB

	revmap map[string]string
	revmut sync.RWMutex
}

func (c *CouchDB) DB() *kivik.DB {
	return c.db
}

func (c *CouchDB) Client() *kivik.Client {
	return c.client
}

func NewCouchDB(dsn string) (*CouchDB, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, err
	}
	database := strings.Trim(u.Path, "/")
	u.Path = ""
	dsn = u.String()

	cdb := &CouchDB{}
	// cdb.Init()

	client, err := kivik.New("couch", dsn)
	if err != nil {
		return nil, err
	}

	const maxTries = 5
	tries := 0
	for {
		if exists, err := client.DBExists(context.TODO(), database); err == nil && !exists {
			if err := client.CreateDB(context.TODO(), database); err != nil {
				return nil, err
			}
			break
		} else if err != nil {
			if c := kivik.StatusCode(err); c >= http.StatusBadGateway && c <= http.StatusGatewayTimeout {
				if tries == maxTries {
					log.Errorf("Failed to connect to couch after %d tries, giving up", maxTries)
					return nil, err
				}
				tries++
				wait := time.Duration(3*tries) * time.Second
				log.Warnf("Couch not ready yet, retrying %d: waiting %v...", tries, wait)
				time.Sleep(wait)
			} else {
				return nil, err
			}
		} else {
			// err == nil && exists
			break
		}
	}

	cdb.db = client.DB(context.TODO(), database)
	cdb.client = client

	// Check(cdb.db)
	return cdb, nil
}

func (cdb *CouchDB) Init() {
	cdb.revmap = make(map[string]string)
}

// GetClient allows direct access to the database client, for those who need it
func (cdb *CouchDB) GetClient() *kivik.Client {
	return cdb.client
}

// GetDB allows direct access to the database, for those who need it
func (cdb *CouchDB) GetDB() *kivik.DB {
	return cdb.db
}

func (cdb *CouchDB) StoreRev(docId, rev string) {
	defer cdb.revmut.Unlock()
	cdb.revmut.Lock()
	cdb.revmap[docId] = rev
}

func (cdb *CouchDB) GetRev(docId string) string {
	defer cdb.revmut.RUnlock()
	cdb.revmut.RLock()
	// returning empty string if not found is what we want
	return cdb.revmap[docId]
}

// Test helper
