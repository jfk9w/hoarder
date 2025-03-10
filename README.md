## hoarder

[![Go Coverage](https://github.com/jfk9w/hoarder/wiki/coverage.svg)](https://raw.githack.com/wiki/jfk9w/hoarder/coverage.html)
[![CodeQL](https://github.com/jfk9w/hoarder/workflows/CodeQL/badge.svg)](https://github.com/jfk9w/hoarder/actions?query=workflow%3ACodeQL)

Бот для сбора данных.

### Описание

Бот предоставляет возможность сбора данных из внешних систем с помощью джобов
и интерфейс для их старта с помощью триггеров.

### Конфигурация

Конфигурация может осуществляться через переменные среды, конфигурационные файлы и просто
через передачу CLI-аргументов.

Самое простое – конфигурация через JSON-файл. Схему для конфигурации можно посмотреть 
[здесь](https://github.com/jfk9w/hoarder/blob/master/config/schema.yaml). Включение и конфигурация
тех или иных джобов и триггеров осуществляется через задание соответствующей секции в файле
конфигурации.

### Запуск

#### Docker (рекомендуется)

```bash
docker -v ${PWD}/config.json:/config.json:ro -v /dev/shm:/dev/shm ghcr.io/jfk9w/hoarder:master --config.file=/config.json
```

#### Сборка из исходников

Необходим Go версии не ниже 1.22.

```bash
git clone git@github.com:jfk9w/hoarder.git
cd hoarder
make build
bin/hoarder --config.file=config.json
```

### Джобы

Реализуют логику инкрементального или полного извлечения данных из 
внешней системы, преобразования их и сохранения в некоторое хранилище.

Для джобов, где необходимо подтверждение пользователя (по коду из СМС, например)
реализована конфигурация авторизационных данных для каждого пользователя. Соответствующий
пользователь должен быть задан в конфигурации триггера, которым он может использовать
(если применимо).

#### lkdr

Чеки и фискальные данные из сервиса ФНС "Мои чеки онлайн". Для использования необходимо хотя бы
раз авторизоваться на сайте с нужным номером телефона.

**Авторизация**

Обратите внимания, что для выполнения авторизации требуется токен капчи. Его можно получить в автоматическом
режиме через сервис [RuCaptcha](https://rucaptcha.com), задав API-ключ в соответствующей секции конфигурации.
Потребуется ввести код подтверждения из СМС.

Альтернатива – подсмотреть токены в браузере и руками внести их в базу.
В этом случае нужно соответственно заполнить `deviceId` и `userAgent` в настройках пользователя в конфигурации джобы.

Токены вечные (по крайней мере пока), поэтому авторизацию нужно выполнить только один раз.

**Поддерживаемые базы данных**

* `sqlite`
* `postgres`
* `mysql` (не протестировано)

#### tinkoff

Счета, выписки, операции, чеки, инвестиционные счета и операции из онлайн-банка "Тинькофф".

**Авторизация**

Авторизация происходит с помощью Selenium. Для работы потребуется `chromedriver` (для Chrome/Chromium) или `geckodriver` (для Firefox), `selenium-server-standalone-*.jar` (можно взять [отсюда](https://selenium-release.storage.googleapis.com/index.html), протестировано с версией 3.14.0), JRE и задание соответствующей конфигурации в секции `selenium`. 
В Docker-образе необходимое ПО уже установлено, дополнительная конфигурация не требуется (используется `geckodriver`).

Для авторизации потребуется ввести код подтверждения из СМС.

Токен живет примерно сутки при условии регулярного пинга (выполняется клиентом автоматически).

**Поддерживаемые базы данных**

* `sqlite`
* `postgres`
* `mysql` (не протестировано)

#### firefly

Для банковских данных, выгруженных с помощью джобов (на текущий момент только `tinkoff`) есть опция синхронизации
с [Firefly III](https://www.firefly-iii.org/). Синхронизация будет выполняться автоматически после каждого инстанса
соответствующей джобы, если в конфигурации заполнена соответствующая секция.


### Триггеры

| Триггер  | Описание |
|----------|----------|
| telegram | Общение с пользователем (включая запрос кода подтверждения) происходит через Telegram. Для запуска джобов нужно отправить соответствующую команду боту (см. список команд в боте) |
| xmpp     | Общение с пользователем (включая запрос кода подтверждения) происходит по протоколу XMPP (Jabber). Для запуска джобов нужно отправить соответствующее сообщение боту:<br>* `all` для запуска всех джобов<br>* `tinkoff lkdr` для запуска конкретных джобов  |
| schedule | Фоновый запуск джобов с заданным интервалом. Запуск происходит одновременно для всех пользователей, указанных в конфигурации триггера. Коды подтверждения запрашиваться не будут.                                                                           |
| stdin    | Интерактивная командная строка. Рекомендуется отключить триггер `schedule` или перенаправить `stderr` в файл, чтобы логирование не мешало эксплуатации. Для запуска джобов ввести имя нужного пользователя и ID нужных джобов (аналогично триггеру `xmpp`). |

### Ответственность

Проект активно разрабатывается и дополняется. Отсутствие поломок существующего функционала и сохранения совместимости конфигурации не гарантировано.

Обратите внимание, что для конфигурации некоторых джобов необходимы чувствительные данные (логины/пароли).
Примите меры для защиты конфигурационных файлов от доступа третьими лицами.
