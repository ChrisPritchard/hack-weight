# Hack Weight: A weight and calorie tracking website/app

A web app designed to be accessed using mobiles for tracking weight and calorie consumption. Based on the [Hacker Diet](https://www.fourmilab.ch/hackdiet/e4/) methodology.

## Database

Hack Weight uses a SQLite3 database, that can be created with the following SQL:

```sql
PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;
CREATE TABLE settings ( setting_key string primary key, setting_value string not null );
CREATE TABLE weight_entry ( id integer primary key, date string not null, weight real not null );
CREATE TABLE calorie_entry ( id integer primary key, date string not null, amount integer not null, category string not null );
COMMIT;
```

## Rationale

I've always struggled with weight, largely because I love good food and good beer, and especially pubs that provide both. I also love pizza, which no doubt doesn't help, and the city I live in literally runs a gourmet burger month every year where the challenge is to try as many different, large and rich burgers as you can. Oh, and they also run a pretty good craft beer festival. Add to that my penchant for spending 95% of my waking hours in a chair in front of a screen and...you get an average BMI of 'Obese'.

Around 10 years prior to creating this, I found the Hacker Diet (discovered while browsing Deus Ex text transcriptions from [nuwen.net, the personal blog site of Stephan T. Lavavej](https://nuwen.net)). Giving it a go, I dropped from 105kg down to 82kg, a weight I managed to keep for five years or so until I met my Wife. She shared my preference for good food, beer and company (though she wasn't much of a beer person prior) and we both ballooned. Especially when our dog and then our daughter came alone. Stress relieve through technical alcoholism is a thing people.

So I'm doing the diet again, and I thought this time I would knock together an app to support it, in my new favourite language Go. Any excuse, right?
