# Node Authoring Projection 3.1 research

## Research Read

- Question: how should Yotta derive node-editor controls, type hints, defaults, constraints, lifecycle, and security requirements without making the frontend a second contract interpreter?
- Target surface: the desktop workflow canvas and Node Inspector, especially schema-driven configuration for 3.1 nodes.
- User job: understand what a node consumes and produces, configure it safely, and see why it needs a target, credential, or runtime-only value.
- Failure to avoid: guessed ports, implicit control-flow pins, silently materialized defaults, tooltip-only explanations, and UI adapters that become semantic authorities.
- Constraints: dense dark desktop UI, keyboard and screen-reader accessibility, generated documentation, strict reopening, and future third-party node packages.

## Source Matrix

| Source | Useful guidance | Yotta application |
| --- | --- | --- |
| [JSON Schema annotations](https://json-schema.org/understanding-json-schema/reference/annotations) | `title`, `description`, `default`, `examples`, `deprecated`, `readOnly`, and `writeOnly` are annotations. A default is not automatic value insertion. | Project safe annotations into explicit UI facts. Keep `hasDefault` separate and never mutate node config merely because a default exists; reject `writeOnly` config because Yotta secrets use host credential slots. |
| [Primer FormControl](https://www.primer.style/product/components/form-control/) and [accessibility guidance](https://www.primer.style/product/components/form-control/accessibility/) | Labels, captions, and validation messages have different jobs and must be associated with the control. | Render a visible label and caption, use stable IDs plus `aria-describedby`, and reserve validation text for actual invalid state. |
| [Primer forms pattern](https://www.primer.style/product/ui-patterns/forms/) | Forms should be grouped, concise, and progressively disclose supporting information. | Keep the primary field editor compact; show lifecycle, permissions, and constraints as structured supporting facts rather than a wall of raw contract JSON. |
| [WAI-ARIA APG: accessible names and descriptions](https://www.w3.org/WAI/ARIA/apg/practices/names-and-descriptions/) | Accessible names should be concise; longer visible descriptions can be referenced with `aria-describedby`. | Do not rely on icon titles or hover tooltips for type and constraint hints. |
| [VS Code configuration contribution point](https://code.visualstudio.com/api/references/contribution-points) | A schema can drive type-specific controls, ordering, enum descriptions, defaults, and descriptions; complex values require an explicit JSON editing experience. | Use scalar controls when projection facts are sufficient. Route unsupported or complex structures to a JSON control or named Editor Adapter instead of guessing. |
| [Carbon tag usage](https://carbondesignsystem.com/components/tag/usage/) | Tags communicate classification or state and should remain concise. | Use compact badges for availability, value lifecycle, and capability risk; keep operational details in visible text. |

## Patterns

1. Generate one complete Authoring Projection from exact Data Type, Node Contract, Capability, and Catalog references. The projection is a cache, never a semantic authority.
2. Reopen projections by regenerating from trusted contracts and requiring byte equality. This prevents edited UI metadata from changing execution semantics.
3. Project scalar controls directly: string, number, integer, boolean, and enum. Use explicit JSON or a named Editor Adapter for complex values.
4. Treat schema defaults as hints. Absence remains absence until the user edits the field.
5. Keep field label, caption, constraints, and validation distinct. Every descriptive region receives a stable ID referenced by the control.
6. Show data-port binding and carrier separately from type lifecycle. A durable type can still travel through a runtime resource lease, and a mixed type needs compile-time carrier resolution.
7. Show target, credential, consent, and risk requirements from Capability contracts. Reject `writeOnly` config fields: secret material must use a host credential slot, never Source config. Do not infer platform availability from icon, category, or implementation name.
8. Render signals only when the Node Contract declares them. A data function such as Concat has two data inputs and one data output, with no synthetic `out` signal.

## Local Application

- `internal/nodeauthoring` owns derivation, completeness checks, strict reopen, and projection schema generation.
- `contracts/node/3.1/builtin-authoring.json` is the generated built-in artifact consumed by the frontend and documentation tooling.
- The frontend imports generated TypeScript shapes and never walks raw Node Contract or JSON Schema documents.
- `NodeAuthoringPanel` owns generic 3.1 configuration and visible contract hints. It emits explicit user edits only and does not materialize defaults.
- Specialized interactions such as region picking, code editing, or AE/UE object selection remain Editor Adapters. They receive projected values but do not define ports, types, permissions, or runtime behavior.

## Next Step

Ship the projection generator and schema, integrate the generic panel into the production Inspector, add strict/golden and UI behavior tests, then make generated node documentation consume the same projection facts.
