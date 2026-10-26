# Mobile robot: a first synthetic recording

This small public engineering example asks whether an existing Signal Scout
declared relation can distinguish two simple Python controllers. It is a
one-dimensional, deterministic toy model with a fixed obstacle, perfect
clearance measurement, fixed-speed kinematics and a deliberately changed stop
threshold. It has no geometry, acceleration, sensors, physics engine, ROS,
C++ components or real robot data. Distances and speed are arbitrary units.
It is not evidence of real-world robot safety, incident reproduction, root
cause discovery, diagnostic advantage or product usefulness.

## Run in native PowerShell

Build the checked-out Signal Scout executable with `task build`. Prepare the
locked Python adapter environment described in
[the adapter README](../../adapters/provenance-python/README.md). The example
defaults to the user's prepared adapter at
`C:\Users\yiche\telemetry-project\adapters\provenance-python\.venv\Scripts\python.exe`;
use `-Python` with another prepared interpreter if needed.

From this example's checkout root:

```powershell
task build
& .\examples\mobile-robot\run.ps1 -Python 'C:\Users\yiche\telemetry-project\adapters\provenance-python\.venv\Scripts\python.exe'
```

The script allocates a new GUID-named directory under
`.cache/mobile-robot`; optionally pass `-Out` with a new absolute path. A
collision or failed prerequisite stops execution, preserving existing artifacts.
Port 4318 must be available; a bind check is not a reservation and the script
only stops its own server process. Server termination is forced after the
acknowledged capture, not a graceful shutdown. Synthetic-only capture is not
a production privacy policy.

For each controller, a fresh database stores twelve probed `command` calls.
The uninstrumented and instrumented output must match exactly. Doctor delivery
and the persisted manifest must both report twelve complete runs with no
losses or unknowns. The script then runs the native `relations` command and
checks the expected outcomes: the reference has five satisfied and zero
violated decisions; the intentionally changed controller has three satisfied
and two violated decisions. Seven decisions in each controller are outside the
rule's near-obstacle condition. The rule is supplied by the example author,
not discovered by Signal Scout.

Each controller directory retains `uninstrumented.json`, `simulation.json`,
`doctor.json`, `completeness.json`, `relations.json`, the SQLite database,
and server/probe logs. The relations file links to actual stored decisions.
A viewer URL applies only to the database in the same directory. For example,
start the viewer for the database you want to inspect, then open the witness
URL from its `relations.json`:

```powershell
& .\bin\signalscout.exe --db 'C:\absolute\path\to\regression\capture.db' serve --listen 127.0.0.1:4318 --retention 0
```

The probe records only the declared Python function boundary. It does not
capture arbitrary native/C++ code, a robot's full state, or causal links.
Python is sufficient for this toy example. A real ROS 2 or C++ application
would need a separate integration for its actual topics, recording format or
native boundaries, followed by validation on those systems. This example does
not require or establish that a C++ plugin is the right design.