# Design

A website that allows me to track my weight, calories and get trends.

How will this look? I have two ideas so far:

- a slick, mobile friendly website with a UI, basically a four page website
    - dashboard (shows today, and trend)
    - enter weight
    - enter calories
    - set goals

- a ascii/terminal style interface accessible presented as plain/text
    - interacted with via API

Could do both? Maybe a pure API, but a hosted vue.js client side website? Not sure I can be bothered getting back into the JS framework lands. Instead could just create a single html page with plain js changing its elements. Then wouldn't require any templating, could just host index.html under static, or serve it for '/'. But would even this require static files, css, js etc.

Maybe push these concerns until later. Translate the pages into an API

/today GET dashboard
/today/weight POST send weight
/today/calories POST send calories
/goals POST set goal