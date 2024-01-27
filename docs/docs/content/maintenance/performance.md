# Performance

listmonk is built to be highly performant and can handle millions of subscribers with minimal system resources.

However, as the Postgres database grows—with a large number of subscribers, campaign views, and click records—it can significantly slow down certain aspects of the program, particularly in counting records and aggregating various statistics. For instance, loading admin pages that do these aggregations can take tens of seconds if the database has millions of subscribers.

- Aggregate counts, statistics, and charts on the landing dashboard.
- Subscriber count beside every list on the Lists page.
- Total subscriber count on the Subscribers page.

However, at that scale, viewing the exact number of subscribers or statistics every time the admin panel is accessed becomes mostly unnecessary. On installations with millions of subscribers, where the above pages do not load instantly, it is highly recommended to turn on the `Settings -> Performance -> Cache slow database queries` option.

## Slow query caching

When this option is enabled, the subscriber counts on the Lists page, the Subscribers page, and the statistics on the dashboard, etc., are no longer counted in real-time in the database. Instead, they are updated periodically and cached, resulting in a massive performance boost. The periodicity can be configured on the Settings -> Performance page using a standard crontab expression (default: `0 3 * * *`, which means 3 AM daily). Use a tool like [crontab.guru](https://crontab.guru) for easily generating a desired crontab expression.
