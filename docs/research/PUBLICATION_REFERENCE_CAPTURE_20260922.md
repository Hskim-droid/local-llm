# Publication-strategy reference capture

Reviewed: 2026-09-22

This file is a short source capture for the publication strategy. It records
the parts of each reference that influence the decision; it is not a claim
that local-llm has the referenced project's capabilities.

## External references

### GitHub repository front door

- URL: <https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-readmes>
- Observed: GitHub treats the README as the first place visitors see and
  expects it to explain usefulness, use, help, and maintainership.
- Short source excerpt: “A README is often the first item a visitor will see
  when visiting your repository.”
- Decision use: keep the repository README as the technical front door, but
  move longer evidence and operating detail into linked documents.

### GitHub Releases and stable links

- URL: <https://docs.github.com/en/repositories/releasing-projects-on-github/about-releases>
- URL: <https://docs.github.com/en/repositories/releasing-projects-on-github/linking-to-releases>
- Observed: releases package deployable iterations, are based on Git tags, and
  can carry release notes and binary assets. `releases/latest` and a direct
  asset URL are stable sharing paths.
- Short source excerpt: “Releases are deployable software iterations.”
- Decision use: binaries and installable packs belong in a versioned GitHub
  Release, not in a narrative post or an unpinned `main` link.

### GitHub citation and archive path

- URL: <https://docs.github.com/en/enterprise-cloud@latest/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-citation-files>
- URL: <https://docs.github.com/en/repositories/archiving-a-github-repository/referencing-and-citing-content>
- Observed: `CITATION.cff` adds a repository citation prompt; GitHub documents
  Zenodo as a way to archive public repositories and issue a DOI per release.
- Short source excerpt: “When you add a `CITATION.cff` file ... a link is
  automatically added to the repository landing page.”
- Decision use: add citation metadata when the framework reaches a stable
  externally reusable checkpoint; do not create a DOI for every experimental
  run.

### GitHub artifact provenance

- URL: <https://docs.github.com/en/actions/concepts/security/artifact-attestations>
- Observed: GitHub artifact attestations connect a released artifact to the
  workflow, repository, environment, commit SHA, and triggering event. The
  guidance is for software artifacts, not ordinary documentation files or
  screenshots.
- Short source excerpt: “People who consume your software can verify where and
  how your software was built.”
- Decision use: consider attestations for installable binaries and packaged
  manifests after a CI-built release lane exists; do not add ceremony to every
  synthetic screenshot.

### BrowserGym: research framework publication shape

- URL: <https://github.com/ServiceNow/BrowserGym/blob/main/README.md>
- Observed: the README places setup, usage, demo, ecosystem, paper, and
  citation beside an explicit warning that the framework is not a consumer
  product. The demo is executable and the benchmark families are named.
- Short source excerpt: “Setup - Usage - Demo - Ecosystem - ... Paper -
  Citation.”
- Decision use: every technical project page should expose the runnable path,
  demo/evidence path, reference lineage, and citation path separately.

### OSWorld-V2: pinned benchmark releases and run provenance

- URL: <https://github.com/xlang-ai/OSWorld-V2/blob/main/benchmark_releases/README.md>
- URL: <https://github.com/xlang-ai/OSWorld-V2/blob/main/README.md>
- Observed: a benchmark release is a compact manifest that pins code, tasks,
  assets, websites, and provider images. Per-run provenance lives separately;
  comparable runs must not mix `main` or `latest` with a release.
- Short source excerpt: “Avoid floating references such as `main` or `latest`
  in comparable runs.”
- Decision use: local-llm evidence packets need an immutable commit/tag plus
  input/output hashes and an explicit run record; a post must never imply that
  an unpinned branch is a reproducible release.

### Reproducible Builds: environment and checksums

- URL: <https://reproducible-builds.org/docs/>
- URL: <https://reproducible-builds.org/docs/recording/>
- Observed: reproducibility requires a recreatable environment, stable inputs
  and outputs, and published build information alongside the artifact.
- Short source excerpt: “Stable inputs. Stable outputs. Capture as little as
  possible from the environment.”
- Decision use: publish the smallest environment record needed to rerun a
  synthetic example, and keep private host identifiers out of public packets.

### Microsoft MarkItDown: input breadth with explicit boundaries

- URL: <https://github.com/microsoft/markitdown/blob/main/README.md>
- URL: <https://github.com/microsoft/markitdown/blob/main/packages/markitdown-mcp/README.md>
- Observed: the project describes supported input families, provides CLI and
  API examples, keeps plugins optional, and documents local privilege and
  localhost exposure risks. It also states that conversion is aimed at text
  analysis rather than perfect human-facing document fidelity.
- Short source excerpt: “Plugins are disabled by default.”
- Decision use: publish the core contract and adapter boundaries first; show
  optional integrations as separately enabled, separately verified extensions.

### OpenHands: repository boundary and docs separation

- URL: <https://github.com/OpenHands/OpenHands>
- URL: <https://github.com/OpenHands/docs>
- Observed: the project README links quickstart, docs, self-hosting, agents,
  and automations. Its repository map assigns behavior to separate repositories
  and the docs repository maintains a unified documentation site.
- Short source excerpt: “The self-hosted developer control center for coding
  agents and automations.”
- Decision use: preserve the two-repository boundary: `local-llm` owns code and
  executable contracts; `dropkit.contents` owns human-facing publication.

### Paperless-ngx: documentation, demo, and safety warning

- URL: <https://github.com/paperless-ngx/paperless-ngx>
- Observed: the README links a demo, features, screenshots, getting started,
  documentation, support, translation, and a clear warning about sensitive
  documents and local hosting.
- Short source excerpt: “A full list of features and screenshots are available
  in the documentation.”
- Decision use: a public project page should offer a safe synthetic demo and a
  direct limitations/privacy statement, not imply that a demo is production
  validation.

## Internal references reviewed

- `local-llm/README.md`
- `local-llm/docs/WORKFLOW_CONTRACT.md`
- `local-llm/docs/RPA_REFERENCE_LINEAGE.md`
- `dropkit.contents/src/pages/work/local-llm.astro`
- `dropkit.contents/src/pages/work/scan-recovery.astro`
- `dropkit.contents/public/examples/scan-recovery/verification.json`
- GitHub release snapshot: `local-llm` latest public binary release `v0.9.4`,
  published 2026-09-03; current `main` at `db39e7d` is 15 commits beyond that
  tag and includes the integrated harness/SDK work.

## Working conclusion

The canonical unit should not be a post, screenshot, or a model transcript.
It should be an **evidence packet**: a bounded claim, pinned source revision,
synthetic or explicitly permitted input, output artifact, verification record,
reproduction command, and limitations. The publication hub may narrate that
packet, but it must link back to the technical source and exact evidence.
