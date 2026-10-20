package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type demoSeller struct {
	username string
	game     string
	title    string
	category string
	details  string
	price    int64
}

var demoSellers = []demoSeller{
	{"aida_gg", "Genshin Impact", "Аккаунт с редкими персонажами", "account", "Подробности аккаунта — в личном чате.", 12500},
	{"temirlan_play", "Counter-Strike 2", "Помощь с настройкой инвентаря", "service", "Помогу подобрать оформление и настроить инвентарь.", 3500},
	{"dana_pixel", "Minecraft", "Набор ресурсов для выживания", "resource", "Подготовленный набор полезных игровых ресурсов.", 1800},
	{"aruzhan_games", "Roblox", "Игровой набор для Roblox", "resource", "Подробности состава и совместимости уточняйте в чате.", 2200},
	{"nursultan_pro", "Dota 2", "Тренировка для начинающих", "service", "Индивидуальная игровая тренировка для новичков.", 4000},
	{"aliya_play", "Valorant", "Помощь с настройкой прицела", "service", "Подберём настройки под твой стиль игры.", 1500},
	{"dias_gamer", "Fortnite", "Косметический набор Fortnite", "resource", "Игровой набор, подробности — в переписке.", 5000},
	{"malika_online", "Genshin Impact", "Ресурсы для прокачки персонажа", "resource", "Набор игровых ресурсов для быстрого старта.", 2700},
	{"sanzhar_x", "PUBG Mobile", "Консультация по настройкам", "service", "Подскажу настройки управления и чувствительности.", 1200},
	{"tomiris_games", "Brawl Stars", "Игровой аккаунт Brawl Stars", "account", "Описание и детали передачи аккаунта в чате.", 8500},
	{"askar_levelup", "Minecraft", "Строительство базы на заказ", "service", "Построю базу по твоей идее на сервере.", 6000},
	{"aliya_moon", "Genshin Impact", "Материалы для прокачки", "resource", "Подбор материалов для выбранного персонажа.", 1900},
	{"bekzat_fps", "Counter-Strike 2", "Тренировка по стрельбе", "service", "Разберём разминку и базовые игровые привычки.", 3200},
	{"sara_quest", "Honkai: Star Rail", "Аккаунт для старта", "account", "Стартовый аккаунт, подробности уточняйте заранее.", 7000},
	{"arman_builds", "Roblox", "Помощь с игровым проектом", "service", "Подскажу идеи и помогу разобраться с механиками.", 4500},
	{"aigerim_rpg", "The Sims 4", "Подборка дополнений и предметов", "resource", "Подробный список содержимого отправлю в чате.", 2800},
	{"miras_moba", "Mobile Legends", "Разбор матча Mobile Legends", "service", "Обсудим решения и стратегию после матча.", 2300},
	{"zhanel_star", "Honkai: Star Rail", "Ресурсы для команды", "resource", "Помогу собрать нужные материалы для команды.", 2100},
	{"ruslan_kz", "PUBG Mobile", "Подсказки для новичков", "service", "Разберём основы и полезные игровые настройки.", 1600},
	{"madina_craft", "Minecraft", "Набор для строительства", "resource", "Материалы и идеи для строительного проекта.", 2400},
	{"ilyas_rank", "Valorant", "Тренировка перед рейтингом", "service", "Практика механики и разбор игровых ситуаций.", 3700},
	{"kamila_arcade", "Brawl Stars", "Набор ресурсов Brawl Stars", "resource", "Игровые ресурсы, условия уточняются в чате.", 1750},
	{"daniyar_dota", "Dota 2", "Разбор героев и ролей", "service", "Помогу освоить выбранную роль и героев.", 2900},
	{"asyl_kawaii", "The Sims 4", "Декор для игрового дома", "resource", "Идеи и материалы для уютного игрового дома.", 2600},
	{"maxim_gamerkz", "Counter-Strike 2", "Помощь с настройкой игры", "service", "Оптимизация базовых игровых настроек.", 1400},
	{"indira_quest", "Genshin Impact", "Игровые материалы Genshin", "resource", "Уточни нужного персонажа — расскажу подробнее.", 3100},
	{"eldar_plays", "Fortnite", "Разбор строительства", "service", "Потренируем базовые приёмы строительства.", 3300},
	{"nazym_online", "Roblox", "Подборка предметов Roblox", "resource", "Актуальный список предметов отправлю по запросу.", 1950},
	{"vlad_pixel", "Mobile Legends", "Игровой аккаунт Mobile Legends", "account", "Подробности аккаунта и передачи обсуждаются лично.", 9200},
	{"kamron_xp", "Minecraft", "Помощь с прохождением", "service", "Совместно разберём сложный этап или задание.", 2000},
}

func SeedDemoMarketplace(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin demo marketplace seed: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	passwordBytes := make([]byte, 32)
	if _, err := rand.Read(passwordBytes); err != nil {
		return fmt.Errorf("generate demo account credentials: %w", err)
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(hex.EncodeToString(passwordBytes)), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash demo account credentials: %w", err)
	}

	for _, seller := range demoSellers {
		displayName := displayName(seller.username)
		_, err := tx.Exec(ctx, `
			INSERT INTO users (id, username, name, email, role, password_hash)
			VALUES (gen_random_uuid(), $1, $2, $3, 'user', $4)
			ON CONFLICT DO NOTHING`,
			seller.username, displayName, seller.username+"@demo.playerook.invalid", string(passwordHash))
		if err != nil {
			return fmt.Errorf("seed demo user %q: %w", seller.username, err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO listings (id, seller_id, title, game, category, description, price_kzt)
			SELECT gen_random_uuid(), users.id, $2, $3, $4, $5, $6
			FROM users
			WHERE users.username = $1
			  AND NOT EXISTS (
				  SELECT 1 FROM listings
				  WHERE listings.seller_id = users.id AND listings.title = $2
			  )`,
			seller.username, seller.title, seller.game, seller.category, seller.details, seller.price)
		if err != nil {
			return fmt.Errorf("seed listing for %q: %w", seller.username, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit demo marketplace seed: %w", err)
	}
	return nil
}

func displayName(username string) string {
	parts := strings.Split(username, "_")
	for index, part := range parts {
		if len(part) > 0 {
			parts[index] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}
