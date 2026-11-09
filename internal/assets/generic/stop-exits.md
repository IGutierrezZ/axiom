**Stops and exits.** A stop is a point where the orchestrator presents what it just produced and waits for a human decision before any further work: every SDD block-review gate except integration (spec, design, tasks, and each role-apply) of a `checkpointed` change. A `continuous` change has no stops. Every stop offers exactly three exits, always in this order:

1. **Iterate:** keep working on the current artifact and present the stop again. There is no limit to the iterations.
2. **Approve, push and stop:** approve the artifact, hand the next phase to another person, present what must be pushed, and stop. The session never starts the next phase.
3. **Approve and continue:** approve the artifact and continue yourself with the next phase.

"Push" in exit 2 means a pull request merged into its target (`prTarget`) before anyone starts the next phase: `axiom branch ensure` on another machine creates the branch from the merged base and does not recover work that is not merged. In the specs repository of a `multirepo` there is no pull request: exit 2 is a commit on the integration branch and a push done by the human.

`push`, `fetch`, opening the pull request, and merging belong to the human: present the exact commands and data, and never run them yourself. With the `engram` or `none` artifact store the artifacts are not files, so exit 2 is not available: offer only exits 1 and 3. The person who picks up a handed-off phase starts the next phase without reviewing the previous one again.
