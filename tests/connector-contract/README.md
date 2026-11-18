# Connector contract testing

One shared suite runs against General's OpenStack connector, the AWS connector and the mock adapter. Patrii Cloud has no connector yet, so its row skips with the reason.

Run it with `bash tests/connector-contract/run.sh`. What it covers, what it needs and where its code lives: [tests/README.md](../README.md).
