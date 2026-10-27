package vesna

import (
	"fmt"

	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/info"
)

const normalHitNum = 6

var (
	attackFrames   [][]int
	attackHitmarks = [][]int{{12}, {12}, {10, 21}, {15}, {14}, {21}}
	// hit lag (Avatar_Girl_Sword_Vesna_AniEvent_Attack)
	attackHitlagHaltFrame = [][]float64{{0.05}, {0.05}, {0.03, 0.03}, {0.02}, {0.03}, {0.08}}
	// hitboxes: boxes [width, length] for N1, N3 and N6, circles otherwise
	attackHitboxes = [][]float64{{2.26, 5.43}, {2.4}, {2.26, 5.43}, {3}, {2.8}, {6, 5}}
	attackOffsets  = []info.Point{{Y: -1}, {Y: 0.98}, {Y: -1.53}, {Y: 0.2}, {Y: 0.5}, {X: -0.43, Y: -0.76}}
)

func init() {
	attackFrames = make([][]int, normalHitNum)

	attackFrames[0] = frames.InitNormalCancelSlice(attackHitmarks[0][0], 28)
	attackFrames[0][action.ActionAttack] = 21

	attackFrames[1] = frames.InitNormalCancelSlice(attackHitmarks[1][0], 30)
	attackFrames[1][action.ActionAttack] = 23

	attackFrames[2] = frames.InitNormalCancelSlice(attackHitmarks[2][1], 40)
	attackFrames[2][action.ActionAttack] = 33

	attackFrames[3] = frames.InitNormalCancelSlice(attackHitmarks[3][0], 34)
	attackFrames[3][action.ActionAttack] = 27

	attackFrames[4] = frames.InitNormalCancelSlice(attackHitmarks[4][0], 36)
	attackFrames[4][action.ActionAttack] = 29

	attackFrames[5] = frames.InitNormalCancelSlice(attackHitmarks[5][0], 60)
	attackFrames[5][action.ActionAttack] = 52
}

// Normal Attack: up to 6 strikes. In Armed for Action they deal Anemo DMG that no other infusion
// overrides, and every hit summons a wind pinion.
func (c *char) Attack(p map[string]int) (action.Info, error) {
	if c.StatusIsActive(c6TransposeKey) {
		return c.transpose(p)
	}

	n := c.NormalCounter
	for i, mult := range attack[n] {
		ai := info.AttackInfo{
			ActorIndex:         c.Index(),
			Abil:               fmt.Sprintf("Normal %v", n),
			AttackTag:          attacks.AttackTagNormal,
			ICDTag:             attacks.ICDTagNormalAttack,
			ICDGroup:           attacks.ICDGroupDefault,
			StrikeType:         attacks.StrikeTypeSlash,
			Element:            attributes.Physical,
			Durability:         25,
			Mult:               mult[c.TalentLvlAttack()],
			HitlagFactor:       0.01,
			HitlagHaltFrames:   attackHitlagHaltFrame[n][i] * 60,
			CanBeDefenseHalted: true,
		}
		// the attack mode is set when the animation starts (Vesna_NormalAttack_AttackMode_Switcher)
		if c.armed() {
			ai.Element = attributes.Anemo
			ai.IgnoreInfusion = true
		}
		ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), attackOffsets[n], attackHitboxes[n][0])
		if len(attackHitboxes[n]) == 2 {
			ap = combat.NewBoxHitOnTarget(c.Core.Combat.Player(), attackOffsets[n], attackHitboxes[n][0], attackHitboxes[n][1])
		}
		first := i == 0
		c.QueueCharTask(func() {
			c.Core.QueueAttack(ai, ap, 0, 0)
			// one wind pinion per attack, with its first hit
			if first {
				c.pinion(fmt.Sprintf("Wind Pinion (N%v)", n+1))
			}
		}, attackHitmarks[n][i])
	}

	defer c.AdvanceNormalIndex()

	return action.Info{
		Frames:          frames.NewAttackFunc(c.Character, attackFrames),
		AnimationLength: attackFrames[n][action.InvalidAction],
		CanQueueAfter:   attackHitmarks[n][len(attackHitmarks[n])-1],
		State:           action.NormalAttackState,
	}, nil
}
