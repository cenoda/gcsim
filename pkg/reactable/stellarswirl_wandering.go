package reactable

// WanderingVortexKey turns on the Wandering Vortex hook. Vodyanitsa's A1 turns Stellar Vortices
// into Wandering Vortices (ElementAbility_StarSwirl_Ice_Gadget, StarSwirl_Ice_Gadget_AlterHandler
// in the game's data): the vortex itself, its damage and its detonation stay the same, and the
// conversion, the Anemo RES shred it applies and the Radiance: Stellar Swirl extension are kept in
// her package. While the key is set in Flags.Custom, every Stellar Swirl emits
// event.OnStellarSwirlVortex with the vortex it created or stacked, after the Radiance: Stellar
// Swirl state is applied and before the vortex can be detonated at 6 stacks. Only Vodyanitsa
// sets the key, so without her no Stellar Swirl code path changes.
const WanderingVortexKey = "wandering-vortex-enabled"
