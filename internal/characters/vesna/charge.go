package vesna

import (
	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/info"
)

var chargeFrames []int

const chargeHitmark = 24

func init() {
	chargeFrames = frames.InitAbilSlice(52)
	chargeFrames[action.ActionAttack] = 48
	chargeFrames[action.ActionSkill] = 40
	chargeFrames[action.ActionBurst] = 40
	chargeFrames[action.ActionDash] = chargeHitmark
	chargeFrames[action.ActionJump] = chargeHitmark
	chargeFrames[action.ActionSwap] = chargeHitmark
}

// Charged Attack: tramples opponents in front of her; Anemo DMG and a wind pinion in Armed for
// Action.
func (c *char) ChargeAttack(p map[string]int) (action.Info, error) {
	ai := info.AttackInfo{
		ActorIndex:       c.Index(),
		Abil:             "Charge",
		AttackTag:        attacks.AttackTagExtra,
		ICDTag:           attacks.ICDTagExtraAttack,
		ICDGroup:         attacks.ICDGroupDefault,
		StrikeType:       attacks.StrikeTypeSlash,
		Element:          attributes.Physical,
		Durability:       25,
		Mult:             charge[c.TalentLvlAttack()],
		HitlagFactor:     0.01,
		HitlagHaltFrames: 0.03 * 60,
	}
	if c.armed() {
		ai.Element = attributes.Anemo
		ai.IgnoreInfusion = true
	}
	ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), info.Point{Y: 0.57}, 3)
	c.QueueCharTask(func() {
		c.Core.QueueAttack(ai, ap, 0, 0)
		c.pinion("Wind Pinion (Charge)")
	}, chargeHitmark)

	return action.Info{
		Frames:          frames.NewAbilFunc(chargeFrames),
		AnimationLength: chargeFrames[action.InvalidAction],
		CanQueueAfter:   chargeHitmark,
		State:           action.ChargeAttackState,
	}, nil
}
