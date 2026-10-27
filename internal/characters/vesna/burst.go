package vesna

import (
	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/info"
)

var burstFrames []int

const (
	// Avatar_Vesna_ElementalBurst creates SkillObj_Vesna_MagicSword_Big_03 5 m ahead at the press;
	// it strikes on its first think, 0.55s later
	burstHitmark = 33
	burstCD      = 15 * 60
)

func init() {
	burstFrames = frames.InitAbilSlice(100)
	burstFrames[action.ActionAttack] = 92
	burstFrames[action.ActionSkill] = 92
	burstFrames[action.ActionDash] = 90
	burstFrames[action.ActionJump] = 90
	burstFrames[action.ActionSwap] = 88
}

// For the Tsaritsa!: summons Spirit Blades dealing AoE Anemo DMG, or Stellar Swirl DMG in Radiance:
// Stellar Swirl, and gains a stack (30) of Sword Essence. Counts for Disciplinary Action (A1).
func (c *char) Burst(p map[string]int) (action.Info, error) {
	c.addEssence(essenceBurst, "burst")
	c.a1Add()

	// the gadget copies the Radiance state when it appears
	star := c.radiance()
	c.Core.Tasks.Add(func() {
		c.spiritBlade("For the Tsaritsa! Spirit Blade", star, burst[c.TalentLvlBurst()], burstSSw[c.TalentLvlBurst()],
			combat.NewCircleHitOnTarget(c.Core.Combat.Player(), info.Point{Y: 5}, 6.5), attacks.AttackTagElementalBurst)
	}, burstHitmark)

	c.SetCD(action.ActionBurst, burstCD)
	c.ConsumeEnergy(5)

	return action.Info{
		Frames:          frames.NewAbilFunc(burstFrames),
		AnimationLength: burstFrames[action.InvalidAction],
		CanQueueAfter:   burstFrames[action.ActionSwap],
		State:           action.BurstState,
	}, nil
}
