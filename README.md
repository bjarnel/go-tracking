# go-tracking

A very simple tracking service written in GoLang. Storing data in SQLite.

If you need to cross compile this from MacOS to Linux you will need something like musl-cross.

Currently there is no validation, it accepts everything.

Some things I plan to add:

- The option to validate requests
- More stats output options


## Posting events

curl --header "Content-Type: application/json" \
--request POST \
--data "[{\"property\":\"someproperty\",\"ip\":\"192.168.0.0\",\"user_agent\":\"secret agent\",\"description\":\"awesome thing\"}]" \
http://localhost:8091/events

## Fetching stats

curl "http://localhost:8091/stats?property=someproperty"
