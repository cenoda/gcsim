package lohen

import (
	"fmt"

	"github.com/genshinsim/gcsim/internal/frames"
	"github.com/genshinsim/gcsim/pkg/core/action"
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/combat"
	"github.com/genshinsim/gcsim/pkg/core/info"
)

// Hit lag, defense halt, hit counts, strike types and hitbox sizes are from the game's ability data.
// Hitmarks and cancels are estimated: N1 -> CA is set so that the N1C loop takes 43 frames
// (see charge.go), the rest follows Flins.
var (
	attackFrames          [][]int
	attackHitmarks        = [][]int{{11}, {11}, {12, 18, 24}, {20}, {18, 29}}
	attackHitlagHaltFrame = [][]float64{{0}, {0.02}, {0, 0, 0.02}, {0}, {0.02, 0.03}}
	attackDefHalt         = [][]bool{{false}, {true}, {false, false, false}, {false}, {false, true}}
	attackStrikeTypes     = [][]attacks.StrikeType{
		{attacks.StrikeTypeSpear},
		{attacks.StrikeTypeSpear},
		{attacks.StrikeTypeSlash, attacks.StrikeTypeSlash, attacks.StrikeTypeSlash},
		{attacks.StrikeTypeDefault},
		{attacks.StrikeTypeSlash, attacks.StrikeTypeSlash},
	}
	attackHitboxes = [][][]float64{{{2.8}}, {{2.2, 3.5}}, {{2.7, 2.8}, {2.5, 3.1}, {2.2, 2.7}}, {{2.8}}, {{2.2, 3.8}, {2.8}}}
)

const normalHitNum = 5

const (
	joySrcNormal = iota
	joySrcChargeA
	joySrcChargeB
)

func init() {
	attackFrames = make([][]int, normalHitNum)

	attackFrames[0] = frames.InitNormalCancelSlice(attackHitmarks[0][0], 31)
	attackFrames[0][action.ActionAttack] = 14
	attackFrames[0][action.ActionCharge] = 16

	attackFrames[1] = frames.InitNormalCancelSlice(attackHitmarks[1][0], 31)
	attackFrames[1][action.ActionAttack] = 16
	attackFrames[1][action.ActionCharge] = 27

	attackFrames[2] = frames.InitNormalCancelSlice(attackHitmarks[2][2], 48)
	attackFrames[2][action.ActionAttack] = 32
	attackFrames[2][action.ActionCharge] = 34

	attackFrames[3] = frames.InitNormalCancelSlice(attackHitmarks[3][0], 55)
	attackFrames[3][action.ActionAttack] = 37
	attackFrames[3][action.ActionCharge] = 37

	attackFrames[4] = frames.InitNormalCancelSlice(attackHitmarks[4][1], 68)
	attackFrames[4][action.ActionWalk] = 58
}

func (c *char) Attack(p map[string]int) (action.Info, error) {
	masterstroke := c.masterstrokeActive()
	for i := range attackHitmarks[c.NormalCounter] {
		ai := info.AttackInfo{
			ActorIndex:         c.Index(),
			Abil:               fmt.Sprintf("Normal %v", c.NormalCounter),
			AttackTag:          attacks.AttackTagNormal,
			ICDTag:             attacks.ICDTagNormalAttack,
			ICDGroup:           attacks.ICDGroupLohenNormalAttack,
			StrikeType:         attackStrikeTypes[c.NormalCounter][i],
			Element:            attributes.Physical,
			Durability:         25,
			Mult:               attack[c.NormalCounter][i][c.TalentLvlAttack()],
			HitlagFactor:       0.01,
			HitlagHaltFrames:   attackHitlagHaltFrame[c.NormalCounter][i] * 60,
			CanBeDefenseHalted: attackDefHalt[c.NormalCounter][i],
		}
		if masterstroke {
			ai.Abil = fmt.Sprintf("Masterstroke Normal %v", c.NormalCounter)
			ai.Element = attributes.Cryo
			ai.Mult = skillAttack[c.NormalCounter][i][c.skillLvl()]
		}
		hitbox := attackHitboxes[c.NormalCounter][i]
		ap := combat.NewCircleHitOnTarget(c.Core.Combat.Player(), nil, hitbox[0])
		if len(hitbox) == 2 {
			ap = combat.NewBoxHitOnTarget(c.Core.Combat.Player(), nil, hitbox[0], hitbox[1])
		}
		c.QueueCharTask(func() {
			c.Core.QueueAttack(ai, ap, 0, 0, c.joyCB(joySrcNormal), c.particleCB, c.c2CB)
		}, attackHitmarks[c.NormalCounter][i])
	}

	defer c.AdvanceNormalIndex()

	return action.Info{
		Frames:          frames.NewAttackFunc(c.Character, attackFrames),
		AnimationLength: attackFrames[c.NormalCounter][action.InvalidAction],
		CanQueueAfter:   attackHitmarks[c.NormalCounter][len(attackHitmarks[c.NormalCounter])-1],
		State:           action.NormalAttackState,
	}, nil
}

func (c *char) joyCB(src int) info.AttackCBFunc {
	return func(a info.AttackCB) {
		if a.Target.Type() != info.TargettableEnemy {
			return
		}
		c.addJoy(src)
	}
}
