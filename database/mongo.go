package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/an1shka/go-project-playerook-kz/models"
)

func ConnectMongo(ctx context.Context, uri, databaseName string) (*mongo.Database, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect to MongoDB: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("ping MongoDB: %w", err)
	}

	db := client.Database(databaseName)
	_, err = db.Collection("messages").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "chat_id", Value: 1}, {Key: "created_at", Value: -1}},
	})
	if err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("create message index: %w", err)
	}
	_, err = db.Collection("news").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "slug", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true),
	})
	if err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("create news index: %w", err)
	}
	_, err = db.Collection("news").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "created_at", Value: -1}},
	})
	if err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("create news date index: %w", err)
	}
	return db, nil
}

func SeedNews(ctx context.Context, db *mongo.Database) error {
	now := time.Now().UTC()
	announcements := []models.News{
		{Slug: "welcome", Title: "Добро пожаловать в Playerook KZ", Body: "Это площадка для игровых объявлений казахстанского сообщества. Ищи аккаунты, ресурсы и услуги в одном каталоге.", CreatedAt: now},
		{Slug: "prices-kzt", Title: "Все цены — в тенге", Body: "Для удобства игроков из Казахстана цены в каталоге отображаются в KZT. Уточняйте детали сделки в личном чате.", CreatedAt: now.Add(-time.Minute)},
		{Slug: "safe-chat", Title: "Обсуждай условия в чате сделки", Body: "Перед покупкой обсудите с продавцом состав товара, способ передачи и итоговую цену. Не отправляйте пароли от личных аккаунтов.", CreatedAt: now.Add(-2 * time.Minute)},
		{Slug: "seller-tips", Title: "Совет продавцам: заполняйте описание", Body: "Укажите игру, платформу, регион и важные характеристики товара. Подробные объявления помогают быстрее найти покупателя.", CreatedAt: now.Add(-3 * time.Minute)},
		{Slug: "buyer-tips", Title: "Совет покупателям: задавайте вопросы", Body: "Проверьте, что именно входит в предложение, попросите актуальные подробности и не переходите по подозрительным ссылкам.", CreatedAt: now.Add(-4 * time.Minute)},
		{Slug: "weekend-sale", Title: "Распродажа выходного дня", Body: "Следи за новыми объявлениями сообщества: продавцы могут добавлять специальные цены и сезонные предложения.", CreatedAt: now.Add(-5 * time.Minute)},
		{Slug: "profile-nickname", Title: "Твой ник — твоя визитка", Body: "Выбери запоминающийся никнейм в профиле. Другие пользователи смогут найти тебя и написать в чат по нику.", CreatedAt: now.Add(-6 * time.Minute)},
		{Slug: "community", Title: "Строим игровое сообщество вместе", Body: "Будь вежлив в переписке, описывай предложения честно и сообщай о проблемах администрации проекта.", CreatedAt: now.Add(-7 * time.Minute)},
	}
	for _, announcement := range announcements {
		_, err := db.Collection("news").UpdateOne(ctx, bson.M{"slug": announcement.Slug},
			bson.M{"$setOnInsert": announcement}, options.Update().SetUpsert(true))
		if err != nil {
			return fmt.Errorf("seed news %q: %w", announcement.Slug, err)
		}
	}
	return nil
}
