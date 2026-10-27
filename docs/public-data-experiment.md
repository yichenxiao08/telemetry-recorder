# Experimental public-data capture

This mode permits an operator to explicitly declare selected measurements from a
public dataset. It does not classify data or verify licensing, consent, privacy,
or dataset identity. The operator must verify the source and select fields.
Private or production data is outside this experiment's supported scope.

Both producer and server require the same policy file bytes and profile. The
selected profile must have a nonempty allowlist consisting only of `number`
rules under `inputs.*`, `output`, or `output.*`. The existing P0 producer and
intake filters still apply. Numeric values, native trace/span IDs and timestamps
are stored; numbers may themselves be sensitive. Logs, process memory, target
stdout/stderr, doctor files, configuration files and existing stores are outside
the filter. Direct CLI writes, unrelated exporters and access by local users are
not controlled. Use only on a trusted workstation with reviewed public data.

The public server rejects existing database files and requires a literal loopback
address. The probe checks the server's mode, selected profile and SHA-256 of the
exact policy bytes before importing the target, and sends the digest with every
batch. A missing or different policy fails closed. This is a consistency check,
not authentication: local processes can spoof or replace a server, and loopback
does not prevent other local users from connecting. HTTP proxies and redirects
are disabled for this producer path. Do not mix other writers into the store.

## Configuration

Create a new policy file, for example `C:\scout-public\policy.json`:

```json
{"version":1,"profiles":{"public":{"fields":{"inputs.speed":{"kind":"number"}}}}}
```

Start the server with an **absolute, nonexistent** database path, a literal
loopback endpoint, and the policy:

```powershell
& 'C:\path\to\signalscout.exe' --db 'C:\scout-public\fresh.db' serve --listen '127.0.0.1:4318' --capture-policy 'C:\scout-public\policy.json' --capture-profile 'public' --public-data-experiment
```

Configure the probe for the same endpoint, policy bytes and profile. A probe's
own capture allowlist must also include each field:

```yaml
version: 1
synthetic_only: false
public_data_experiment: true
endpoint: http://127.0.0.1:4318
capture_policy: C:\scout-public\policy.json
capture_profile: public
probes:
  - module: example
    function: step
    capture:
      args: [speed]
```

Invoke the usual `scout_probe` command with the configuration and the target's
entry point. `auto_otel` must be off. A fresh database path is required each
time the server is started in this mode; remove experimental artifacts using
the operator's own retention procedure. A zero server retention setting disables
automatic cleanup rather than deleting data immediately.

Strict P0 removes decision identity and semantic metadata. Consequently native
relations, invariants, completeness interpretation and reports may be unavailable
or incomplete; a successful capture is not a successful comparison or diagnosis.
The policy does not provide source-of-truth labels, robot replay, real-world
privacy guarantees, or production authorization. Use the original record and
ordinary tooling to check outputs and labels.
