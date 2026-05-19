// frontend/src/components/containers/nodeRegistry/specs/index.ts
// Side-effect imports — pulling this barrel registers all 70 v4 node kinds.
// Mirrors backend internal/services/container/nodekind/specs/specs.go pattern.
import './control'
import './variables'
import './expr'
import './purefunc'
import './detect'
import './input'
import './system'
