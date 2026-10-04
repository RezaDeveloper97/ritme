{{--
    RSS 2.0 of the magazine (L4-04, FeedController). $channel [title, link, self, description, lastBuildDate],
    $items list of [title, link, description, category, pubDate, enclosure: ?[url, length, type]]. Every URL is absolute.
--}}
{!! '<'.'?xml version="1.0" encoding="UTF-8"?'.'>' !!}
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
<channel>
<title>{{ $channel['title'] }}</title>
<link>{{ $channel['link'] }}</link>
<description>{{ $channel['description'] }}</description>
<language>fa-IR</language>
<lastBuildDate>{{ $channel['lastBuildDate'] }}</lastBuildDate>
<ttl>60</ttl>
<atom:link href="{{ $channel['self'] }}" rel="self" type="application/rss+xml"/>
@foreach ($items as $item)
<item>
<title>{{ $item['title'] }}</title>
<link>{{ $item['link'] }}</link>
<guid isPermaLink="true">{{ $item['link'] }}</guid>
<pubDate>{{ $item['pubDate'] }}</pubDate>
@if ($item['category'])
<category>{{ $item['category'] }}</category>
@endif
<description>{{ $item['description'] }}</description>
@if ($item['enclosure'])
<enclosure url="{{ $item['enclosure']['url'] }}" length="{{ $item['enclosure']['length'] }}" type="{{ $item['enclosure']['type'] }}"/>
@endif
</item>
@endforeach
</channel>
</rss>
