# Contributing

This repository accepts bounded improvements to the experimental harness.

- Keep fixtures synthetic and local.
- Do not add real credentials, cookies, ERP/QMS records, customer data, or mail
  addresses.
- Preserve read/search/draft-first behavior and the explicit output format.
- Do not add unattended writes, external senders, or cloud fallbacks as test
  conveniences.

Run the relevant checks before opening a pull request:

```bash
python3 -m pip install -r requirements-fixture.txt
python3 -m playwright install chromium
python3 -m unittest discover -s tests -p 'test_*.py'
python3 -m compileall -q .
git diff --check
```

Describe unverified host, model, or delivery limitations in the pull request.
