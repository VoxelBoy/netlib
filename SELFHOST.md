# Self-hosted signaling downstream

This is VoxelBoy's small server fork of [poki/netlib](https://github.com/poki/netlib).
The upstream ISC license and Go module path are retained. The browser client is
unchanged; the game continues to consume the official `@poki/netlib` npm package.

## Branches and patches

- `main` mirrors upstream `main`, without downstream patches.
- `codex/selfhost` is the default branch and carries the server adaptations.
- Current upstream base: `386f2ed2602b8d64d46ff06de8ef90ffcdedd9f8` (2026-10-24).

The downstream commits replace Cloudflare credential generation with local coturn
TURN REST credentials, allow an array of TURN URLs, restrict signaling to the game's
UUID and configured origins, bound connections and packet rates, require hello
before credentials, discard client analytics events, and add a standalone image.
These preserve the already-tested Loot Hole self-hosting behavior. `TURN_SECRET`,
`TURN_URLS` and `ALLOWED_ORIGINS` are runtime environment variables; no service
secrets, certificates, game source, database contents or deployment credentials
belong in this public fork. The trusted proxy must overwrite `X-Real-IP` and the
signaling listener must not be directly exposed.

The game repository owns host configuration, compose, coturn, deployment, rollback,
browser/device probes and the exact fork revision it consumes. Its preparation
script checks out that immutable commit, with no source overlays.

## Review an upstream update

Start with a clean checkout. Do not advance the deployed server just because
upstream moved. Fetch and inspect the changes first:

```sh
git fetch upstream main
git log --oneline main..upstream/main
git diff main upstream/main -- internal cmd migrations go.mod go.sum lib
git switch main
git merge --ff-only upstream/main
git push origin main
git switch codex/selfhost
git merge --no-ff main
```

Resolve conflicts by preserving the coturn, guard and analytics behavior above.
Review database migrations and rollback requirements. Update the recorded base in
this file. Usually merge the reviewed upstream history together; cherry-pick only
when a specific change needs to be deferred, and record why.

Run formatting, all Go tests (with the race detector where available) and the
image build:

```sh
go test -race ./...
docker build -f Dockerfile.selfhost --build-arg VCS_REF="$(git rev-parse HEAD)" -t netlib-selfhost:check .
```

Push the downstream branch and confirm the `Self-hosted signaling` workflow passes.
Only then update the game's pinned fork commit. Before a deployment, run its
credential rejection, origin/packet limits, relay privacy, reconnect, restart,
reliable/unreliable delivery and multiplayer probes in the isolated test setup.
Carrier, Safari and portal iframe checks remain separate device/platform gates.
An upstream change to `lib/` also needs a browser package/version review and protocol
compatibility checks with the game's native development client.

## CI and deployment

`selfhost.yml` tests and builds only; it publishes no images and deploys nothing.
Upstream's `Build` and `Slack notification` workflows are disabled in this fork's
GitHub settings. Keep them disabled when advancing the mirror branch. The
downstream copies also guard their jobs to the upstream repository. Deployment
remains an explicit operation from the game repository.

Pin the deployed source commit and the built image identity; never deploy a moving
branch or tag implicitly. Keep the previous image and review database compatibility
before rolling back. The original `386f2ed-selfhost` test image can remain in use
until the next authorized deployment.
