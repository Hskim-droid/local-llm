# Contributing

This repository contains two local-first code lanes: the Go/llama.cpp document
engine and the experimental Python UI-to-document harness under
`agent-harness/`.

For the engine:

```bash
cd gramapp
go test ./...
```

For the harness:

```bash
python3 -m pip install -r agent-harness/requirements-fixture.txt
python3 -m playwright install chromium
python3 -m unittest discover -s agent-harness/tests -p 'test_*.py'
python3 -m compileall -q agent-harness
```

Keep fixtures synthetic and local. Do not add credentials, cookies, ERP/QMS
records, customer data, real mail addresses, browser profiles, or source
materials. Preserve the harness read/search/draft-first behavior, explicit
`docx`/`xlsx`/`pptx` output contract, idempotency checks, and fail-closed
permission boundaries. Do not add unattended writes, external senders, or cloud
fallbacks as test conveniences.

Run `git diff --check` before opening a pull request and describe any unverified
host, model, or delivery limitation.
