# Clippan

Clippan is a CouchDB CLI/shell, as an alternative to Fauxton or using `curl` calls.

## Current state

clippan is under develoment. Even in read-only mode (which is the default) it could delete data due to bugs - USE AT YOUR OWN RISK. Do not use on production databases!

The current shell commands may change and most commands are still very minimal with limited error checking.

## Available commands

```
use                   Connect to a database (takes just a database name or full dsn)
databases             List all databases 
createdb              Create a database (disabled, ro mode)
deletedb              Delete a database (disabled, ro mode)
all                   List all docs, paginated 
get                   Get a single document by id 
put                   Create a new document (disabled, ro mode)
edit                  Edit an existing document (disabled, ro mode)
query                 Query a view 
exit                  Exit clippan 
help                  Show help 
```

Effectively, you can create, delete, access databases, create, access and delete documents
in those databases and query views.

## Building, installing

With a recent Go install (>=1.13.x), `make` will build a clippan binary in bin/

There are several pre-compiled releases available in the releases section

Windows is currently not supported because of the dependency on go-prompt, which is not available for windows.

## Invocation

`clippan <dsn> [-c string] [-write]`

`<dsn>` - a full couchdb url optionally including a database, e.g. `http://admin:admin@localhost:5984/mydb`

`-c string` - execute a sequence of `;`-separated commands, e.g `-c "use foo;query -json a b"`

`-write` - start clippan in write mode (read-only is default). Write mode allows creation/deletion of databases and documents

# Thanks

Clippan is powered by [kivik](https://github.com/go-kivik/kivik), an excellent go/couchdb interface, which also inspired the name Clippan (klippan is, like kivik, an Ikea couch)

