# KVSocial Listmonk releases

KVSocial-specific changes are maintained on `kvs-main`. Feature branches are
merged into `kvs-main` through pull requests. The upstream `master` branch
remains the source used when bringing in new Listmonk releases.

## Publish a KVSocial image

1. Merge the intended changes into `kvs-main`.
2. Pull the latest `kvs-main` locally and run `go test ./...`.
3. Create an annotated tag using the upstream version plus a KVSocial revision:

   ```sh
   git switch kvs-main
   git pull --ff-only origin kvs-main
   git tag -a v6.2.0-kvs.1 -m "KVSocial Listmonk v6.2.0-kvs.1"
   git push origin v6.2.0-kvs.1
   ```

4. Confirm the `KVSocial GHCR release` GitHub Actions workflow succeeds.
5. Confirm the package remains private in the GitHub package settings.
6. Deploy the immutable versioned image:

   ```text
   ghcr.io/kvsocial/listmonk:v6.2.0-kvs.1
   ```

The workflow also updates `ghcr.io/kvsocial/listmonk:kvs-latest`, but production
deployments should use a versioned tag so rollbacks are deterministic.

No Docker Hub credentials are required. GitHub Actions publishes to GHCR with
the repository-scoped `GITHUB_TOKEN`.

## Upgrade to a new upstream release

1. Fetch upstream tags and branches.
2. Create an upgrade branch from `kvs-main`.
3. Merge the new upstream release tag into the upgrade branch.
4. Resolve conflicts while preserving KVSocial-specific commits.
5. Run the full test suite and controlled integration tests.
6. Merge the upgrade pull request into `kvs-main`.
7. Publish a new tag such as `v7.0.0-kvs.1`.

This merge-based flow keeps the full KVSocial history. It does not require
cherry-picking every custom commit for each upstream upgrade.
