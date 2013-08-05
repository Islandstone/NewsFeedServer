# NewsFeedServer

This server is the backbone of my app. It's responsible for periodically
downloading RSS feeds from specified news sources, and make the information in
them available to my app so that it is as small as possible, and compressed. In
addition, when the app requests updates, the server should only return items
that are newer, to keep the updates lightweight. This server currently meets all
of there requirements to a certain degree.
