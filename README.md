# Leprechaun
## A cryptocurrency trading bot based on the Luno API...

## Under Construction.....

Leprechaun is a trading bot that runs in a continous loop that executes a single trade per round. Each trading round is separated by a period of time during which the bot snoozes. Basically, Leprechaun helps automate both technical analysis and trading within the context of an 'infinite' trade loop. Leprechaun is in essence a framework that is designed to be extensible to accomodate trading multiple assets on multiple exchanges. Currently, the [Luno exchange](https://luno.com/en/) is the only supported exchange, but there is likely to be support for other similar exchanges like [Quidax](https://quidax.com) in future. Leprechaun is cross-platform and its UI is built with the wonderful [Gioui library](https://gioui.org).

## Analysis Plugin Framework
Leprechaun takes action based on signals emitted by an arbitrary analysis plugin. The plugin is an opaque technical analysis pipeline defined by the Analysis{} interface that recieves as input past prices of an asset. The size and interval of historical price data provided to the plugin by Leprechaun is defined by the PriceDimensions() method of the Analyzer{} interface. As output, the plugin emits one of a number of signals defined by Leprechaun. For example, an analysis plugin after requesting for the previous prices of an asset for the past 30 hours (or past 30 45-minute periods for that matter), and conducting its technical analysis like price trends, mean reversals, etc. on the data recieved, can then emit the GO_LONG signal, which tells Leprechaun to initiate a long trade. Leprechaun automatically tries to complete all long and short trades based on the defined profit margins defined by the user. The analysis plugin's job is only to conduct technical analysis on historical price data (other data might be provided in future) and then emit a signal based on the results of its analysis. This framework allows an arbitrary analysis pipeline to be plugged in to Leprechaun's endless loop architecture, thus allowing great versatility for end-users who may have other ideas/implementations of technical analysis. This means that users can write their own technical analysis pipeline and plug it into Leprechaun's trading loop.

### Signals
The signals currently defined by Leprechaun are:

GO_LONG - This tells Leprechaun to initiate a long trade i.e. to buy an asset with the hope of later selling it at a higher price. The price at which the asset in question is later sold is defined by the profit margin defined by the user. For example, if after recieving a signal to initiate a long trade for Litecoin at the current price, say #20,000, Leprechaun buys #20,000 worth of Litecoin and records it in its internal ledger. If the user has defined a profit margin of say 5% as at the time the trade was initiated, Lerecahun will only sell the asset when its price has moved 5% above the price at which the asset was purchased i.e. #21,000. Until the long trade is completed the Litecoin purchased is 'locked' by Leprechaun, meaning it cannot be sold in a subsequent trading session. Of course, the user can always sell it themselves on the exchange. However, this will disrupt Leprechauns own internal settings hence it is recommended to use Leprechaun to trade assets you do not want to trade yourself, otherwise resetting Leprechaun each time the assets are taded outside of Leprechaun might be necessary.

GO_SHORT - This tells Leprechaun to initiate a short trade i.e. to sell a fixed amount of an asset at a certain price with the aim of later repurchasing the same exaact amount of that asset at a lower price.

WAIT - This tells Leprechaun to not initiate any trade in the current trading round. This is expected to be emitted when technical analysis is inconclusive, e.g. because there has been no significant change in the price of an asset.

BUY - ????
SELL - ????

#### Profit margin

#### Trade modes

## SADOC DIED!!

So, I'm probably not going to touch this again. If anyone who stumbles upon this feels guided to use it as a basis for their own work or even complete it, Please feel free to do so. This is unlicensed and will remain so. Just make sure others can benefit from it. The code compiles quite okay. Most of the basic functionality is already there. My hope was to upgrade the decision making heuristics of leprechaun to use AI (so it learns over time) instead of basic logic. I tried to get started in [unit 2](https://www.github.com/michaellormann/unit2), you can check that out as well. OR you can just pick whatever piece of code you need... Just do whatever the fuck you want with it :)
