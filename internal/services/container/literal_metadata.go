package container

const portableTargetBindingKey = "Target"

// isLiteralMetadataKey reports config.literal entries that belong to the
// portable package envelope rather than to a runtime data-in pin.
//
// Target nodes keep a logical binding slot beside their hydrated local
// selector fields. The slot deliberately survives aggregateContainer so a
// later save/export can preserve the package identity. It is therefore valid
// literal storage, but it is not part of the node's declared runtime inputs.
func isLiteralMetadataKey(kind, key string) bool {
	if key != portableTargetBindingKey {
		return false
	}
	return kind == "Win32WindowTarget" || kind == "AndroidTarget"
}
