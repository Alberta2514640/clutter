# AWS regression testing

A's whole AWS test suite runs at the recorded source revision in `BASELINE` (before System B was integrated) and at the current tree, and the results are compared test by test.

Run it with `bash tests/aws-regression/run.sh`. What it covers, what it needs and where its code lives: [tests/README.md](../README.md).
