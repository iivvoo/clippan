# Clippan

Clippan is a CouchDB CLI/shell, as an alternative to Fauxton or using `curl` calls.

## Current state

clippan is under develoment. Even in read-only mode (which is the default) it could delete data due to bugs - USE AT YOUR OWN RISK. Do not use on production databases!

The current shell commands may change and most commands are still very minimal with limited error checking.

## Available commands

```
databases             List all databases 
use                   Connect to a database 
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

## Invocation

clippan <dsn> [-c string] [-write]
`<dsn>` - a full couchdb url optionally including a database, e.g. `http://admin:admin@localhost:5984/mydb`
`-c string` - execute a sequence of `;`-separated commands, e.g `-c "use foo;query -json a b"`
`-write` - start clippan in write mode (read-only is default). Write mode allows creation/deletion of databases and documents

# Thanks

Clippan is powered by [kivik](https://github.com/go-kivik/kivik](kivik), an excellent go/couchdb interface, which also inspired the name Clippan (klippan is, like kivik, an Ikea couch)

