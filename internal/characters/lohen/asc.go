package lohen

import (
	"github.com/genshinsim/gcsim/pkg/core/attacks"
	"github.com/genshinsim/gcsim/pkg/core/attributes"
	"github.com/genshinsim/gcsim/pkg/core/event"
	"github.com/genshinsim/gcsim/pkg/core/info"
	"github.com/genshinsim/gcsim/pkg/core/player/character"
	"github.com/genshinsim/gcsim/pkg/enemy"
	"github.com/genshinsim/gcsim/pkg/modifier"
)

const (
	a1Ratio = 30 // hit of at least 3000% of Lohen's Base ATK
	a1Will  = 60

	a4Atk = 0.15
	a4Dur = 8 * 60
	a4Key = "lohen-a4"

	highSpiritsKey      = "lohen-high-spirits"
	highSpiritsICDKey   = "lohen-high-spirits-icd"
	highSpiritsDur      = 9 * 60
	highSpiritsExtraDur = 6 * 60
	highSpiritsICD      = 18 * 60
)

const (
	willGateSuccess = iota
	willGateFail
	willGateA1
)

// Will to Win: while Masterstroke is active, each teammate's damage adds 20 when it is at
// least 1000% of Lohen's Base ATK and 1 otherwise, each at most once every 0.1s per teammate.
// A1 adds another 60 for damage of at least 3000%, on its own 0.1s gate.
// Transformative reaction damage is not counted.
func (c *char) willInit() {
	c.Core.Events.Subscribe(event.OnEnemyDamage, func(args ...any) {
		if _, ok := args[0].(*enemy.Enemy); !ok {
			return
		}
		atk := args[1].(*info.AttackEvent)
		dmg := args[2].(float64)
		if atk.Info.ActorIndex == c.Index() {
			return
		}
		if atk.Info.AttackTag.IsReaction() && !atk.Info.AttackTag.IsDirect() {
			return
		}
		if !c.masterstrokeActive() || dmg <= 0 {
			return
		}
		baseAtk := c.Stat(attributes.BaseATK)
		gate := &c.willGate[atk.Info.ActorIndex]
		tryGate := func(i int, amt float64) {
			if c.Core.F < gate[i] {
				return
			}
			gate[i] = c.Core.F + willGateDur
			c.addWill(amt)
		}
		if dmg >= baseAtk*willRatio {
			tryGate(willGateSuccess, willSuccess)
		} else {
			tryGate(willGateFail, willFail)
		}
		if c.Base.Ascension >= 1 && dmg >= baseAtk*a1Ratio {
			tryGate(willGateA1, a1Will)
		}
	}, "lohen-will-to-win")
}

// A4: while Masterstroke is active, a Cryo reaction triggered by another party member
// gives that member and Lohen 15% ATK for 8s.
func (c *char) a4Init() {
	if c.Base.Ascension < 4 {
		return
	}
	mAtk := make([]float64, attributes.EndStatType)
	mAtk[attributes.ATKP] = a4Atk
	buff := func(char *character.CharWrapper) {
		char.AddStatMod(character.StatMod{
			Base:         modifier.NewBaseWithHitlag(a4Key, a4Dur),
			AffectedStat: attributes.ATKP,
			Amount: func() []float64 {
				return mAtk
			},
		})
	}
	hook := func(args ...any) {
		if _, ok := args[0].(*enemy.Enemy); !ok {
			return
		}
		atk := args[1].(*info.AttackEvent)
		if atk.Info.ActorIndex == c.Index() {
			return
		}
		if !c.masterstrokeActive() {
			return
		}
		buff(c.Core.Player.ByIndex(atk.Info.ActorIndex))
		buff(c.Character.CharWrapper)
	}
	for _, evt := range []event.Event{
		event.OnMelt,
		event.OnFrozen,
		event.OnSuperconduct,
		event.OnSwirlCryo,
		event.OnStellarSwirl,
		event.OnCrystallizeCryo,
		event.OnStellarConduct,
	} {
		c.Core.Events.Subscribe(evt, hook, "lohen-a4")
	}
}

// When the Mood Strikes: entering Masterstroke gives High Spirits (+1 skill level) for 9s,
// 15s when another member's normal attack, skill or burst level is at least Lohen's skill level.
func (c *char) highSpirits() {
	if c.StatusIsActive(highSpiritsICDKey) {
		return
	}
	c.AddStatus(highSpiritsICDKey, highSpiritsICD, true)
	dur := highSpiritsDur
	lvl := c.TalentLvlSkill()
	for _, char := range c.Core.Player.Chars() {
		if char.Index() == c.Index() {
			continue
		}
		if char.TalentLvlAttack() >= lvl || char.TalentLvlSkill() >= lvl || char.TalentLvlBurst() >= lvl {
			dur += highSpiritsExtraDur
			break
		}
	}
	c.AddStatus(highSpiritsKey, dur, true)
}

// Hexerei: Secret Rite. When Etched Into Bone and Soul hits or the burst is unleashed with
// Will to Win at 50% of its limit or more, normal and charged attack DMG +40% for 6s.
func (c *char) hexerei(will float64) {
	if !c.IsHexerei || c.Core.Player.GetHexereiCount() < 2 {
		return
	}
	if will < c.willMax()*hexThreshold {
		return
	}
	m := make([]float64, attributes.EndStatType)
	m[attributes.DmgP] = hexBonus
	c.AddAttackMod(character.AttackMod{
		Base: modifier.NewBaseWithHitlag(hexKey, hexDur),
		Amount: func(atk *info.AttackEvent, _ info.Target) []float64 {
			switch atk.Info.AttackTag {
			case attacks.AttackTagNormal, attacks.AttackTagExtra:
				return m
			}
			return nil
		},
	})
}
