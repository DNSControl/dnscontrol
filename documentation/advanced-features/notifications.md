# Notifications

DNSControl's "notifications" feature will log `push` changes to other services in real time. Typically this is used to automatically announce DNS changes in a team chatroom.  The functionality is implemented using the open source [Shoutrrr](https://github.com/nicholas-fedor/shoutrrr) library, which knows how to communicate to many different systems.  Some additional services are provided natively, see the [notifications package](https://github.com/DNSControl/dnscontrol/tree/main/pkg/notifications).

## Configuration

Notifications are configured in the `creds.json` file, since they often contain API keys or other secrets. The `notifications` key lists the notification service and options.

{% code title="creds.json" %}
```json
{
  "r53": {},
  "gcloud": {},
  "notifications": {
    "notify_on_push": false,
    "notify_on_preview": false,
    "slack_url": "https://api.slack.com/apps/0XXX0X0XX0/incoming-webhooks",
    "teams_url": "https://outlook.office.com/webhook/00000000-0000-0000-0000-000000000000@00000000-0000-0000-0000-000000000000/IncomingWebhook/00000000000000000000000000000000/00000000-0000-0000-0000-000000000000",
    "shoutrrr_url": "discover://token@id"
  }
}
```
{% endcode %}

## Usage

There are two ways to enable notifications:

1. **On-demand**: Add the `--notify` flag to `dnscontrol preview` or `dnscontrol push` commands
2. **Automatic**: Enable `notify_on_push` or `notify_on_preview` in your `creds.json`

### Automatic notifications

To automatically send notifications without using the `--notify` flag, configure these options in your `creds.json`:

- `notify_on_push`: Set to `true` to send notifications during `dnscontrol push` (when changes are applied)
- `notify_on_preview`: Set to `true` to send notifications during `dnscontrol preview` (dry-run mode)

**Example: Notifications only for actual changes (push)**

{% code title="creds.json" %}
```json
{
  "notifications": {
    "notify_on_push": true,
    "notify_on_preview": false,
    "slack_url": "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
  }
}
```
{% endcode %}

**Example: Notifications for both preview and push**

{% code title="creds.json" %}
```json
{
  "notifications": {
    "notify_on_push": true,
    "notify_on_preview": true,
    "teams_url": "https://outlook.office.com/webhook/..."
  }
}
```
{% endcode %}

With automatic notifications enabled, you don't need to use the `--notify` flag. The `--notify` flag will still work to send notifications on-demand regardless of these settings.

### Example DNS change

Below is an example where we add [the A record](../language-reference/domain-modifiers/A.md) `foo` and display the notification output.

{% code title="dnsconfig.js" %}
```diff
D("example.com", REG_MY_PROVIDER, DnsProvider(DSP_MY_PROVIDER),
+ A("foo", "1.2.3.4"),
);
```
{% endcode %}

### Preview example

Preview notifications show what changes would be made without applying them:

{% tabs %}

{% tab title="With --notify flag" %}
```shell
dnscontrol preview --notify
```
{% endtab %}

{% tab title="Automatically" %}
```shell
# Requires notify_on_preview: true in creds.json
dnscontrol preview
```
{% endtab %}

{% endtabs %}

**The notification output**

```shell
**Preview: example.com[my_provider] -** CREATE foo.example.com A (1.2.3.4 ttl=86400)
```

### Push example

Push notifications show the actual changes being applied:

{% tabs %}

{% tab title="With --notify flag" %}
```shell
dnscontrol push --notify
```
{% endtab %}

{% tab title="Automatically" %}
```shell
# Requires notify_on_push: true in creds.json
dnscontrol push
```
{% endtab %}

{% endtabs %}

**The notification output**

```shell
Successfully ran correction for **example.com[my_provider]** - CREATE foo.example.com A 1.2.3.4 ttl=86400
```

## Notification services

### Shoutrrr

DNSControl supports various notification methods via Shoutrrr, including email (SMTP), Discord, Pushover, and many others. For detailed setup instructions, click on the desired service:

#### Chat and Messaging Platforms
 * [Discord](https://shoutrrr.nickfedor.com/v0.12.0/services/chat/discord/)
 * [Google Chat](https://shoutrrr.nickfedor.com/v0.12.0/services/chat/googlechat/) (formerly Hangouts)
 * [Lark](https://shoutrrr.nickfedor.com/v0.12.0/services/chat/lark/)
 * [Matrix](https://shoutrrr.nickfedor.com/v0.12.0/services/chat/matrix/)
 * [Mattermost](https://shoutrrr.nickfedor.com/v0.12.0/services/chat/mattermost/)
 * [Rocket.Chat](https://shoutrrr.nickfedor.com/v0.12.0/services/chat/rocketchat/)
 * [Signal](https://shoutrrr.nickfedor.com/v0.12.0/services/chat/signal/)
 * [Slack](https://shoutrrr.nickfedor.com/v0.12.0/services/chat/slack/)
 * [Teams](https://shoutrrr.nickfedor.com/v0.12.0/services/chat/teams/)
 * [Telegram](https://shoutrrr.nickfedor.com/v0.12.0/services/chat/telegram/)
 * [WeCom](https://shoutrrr.nickfedor.com/v0.12.0/services/chat/wecom/)
 * [Zulip](https://shoutrrr.nickfedor.com/v0.12.0/services/chat/zulip/)
#### Push Notification Services
 * [Bark](https://shoutrrr.nickfedor.com/v0.12.0/services/push/bark/)
 * [Gotify](https://shoutrrr.nickfedor.com/v0.12.0/services/push/gotify/)
 * [IFTTT](https://shoutrrr.nickfedor.com/v0.12.0/services/push/ifttt/)
 * [Join](https://shoutrrr.nickfedor.com/v0.12.0/services/push/join/)
 * [Ntfy](https://shoutrrr.nickfedor.com/v0.12.0/services/push/ntfy/)
 * [Pushbullet](https://shoutrrr.nickfedor.com/v0.12.0/services/push/pushbullet/)
 * [Pushover](https://shoutrrr.nickfedor.com/v0.12.0/services/push/pushover/)
#### Incident and Alert Management
 * [OpsGenie](https://shoutrrr.nickfedor.com/v0.12.0/services/incident/opsgenie/)
 * [PagerDuty](https://shoutrrr.nickfedor.com/v0.12.0/services/incident/pagerduty/)
#### Email Services
 * [SMTP](https://shoutrrr.nickfedor.com/v0.12.0/services/email/smtp/)
#### Specialized Services
 * [Generic](https://shoutrrr.nickfedor.com/v0.12.0/services/specialized/generic/)
 * [Notifiarr](https://shoutrrr.nickfedor.com/v0.12.0/services/specialized/notifiarr/)

Configure `shoutrrr_url` with the Shoutrrr URL to be notified.

{% code title="creds.json" %}
```json
{
  "notifications": {
    "shoutrrr_url": "discover://token@id"
  }
}
```
{% endcode %}

### Slack/Mattermost

To use the Slack integration, you need to create a webhook in Slack.
Please see the [Slack documentation](https://api.slack.com/messaging/webhooks) or the [Mattermost documentation](https://docs.mattermost.com/developer/webhooks-incoming.html)

Configure `slack_url` to this webhook. Mattermost works as well, as they share the same api,

### Microsoft Teams

To use the Teams integration, you need to create a webhook in Teams.
Please see the [Teams documentation](https://docs.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook#add-an-incoming-webhook-to-a-teams-channel)

Configure `teams_url` to this webhook.

### Telegram

To use the [Telegram](https://telegram.org/) integration, you need to create a Telegram bot and obtain a Bot Token, as well as a Chat ID. Get a Bot Token by contacting [@BotFather](https://telegram.me/botfather), and a Chat ID by contacting [@myidbot](https://telegram.me/myidbot).

Configure `telegram_bot_token` and `telegram_chat_id` to these values.

### Bonfire

This is Stack Overflow's built in chat system. This is probably not useful for most people.

Configure `bonfire_url` to be the full url including room and api key.
