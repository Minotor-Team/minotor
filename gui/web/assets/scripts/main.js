/* global Stimulus, Viz */
/* exported main */

// main is the entry point of the script, called when the DOM is loaded
//
// This script relies on the stimulus library to bind data between the HTML and
// the javascript. In short each HTML element with a special
// `data-XXX-target=yyy` attribute can be accessed in the corresponding `XXX`
// controller as the `this.yyyTarget` variable. See
// https://stimulus.hotwired.dev/ for a complete introduction.

const main = function () {
    const application = Stimulus.Application.start();

    application.register("flash", Flash);
    application.register("peerInfo", PeerInfo);
    application.register("messaging", Messaging);
    application.register("unicast", Unicast);
    application.register("routing", Routing);
    application.register("packets", Packets);
    application.register("broadcast", Broadcast);
    application.register("catalog", Catalog);
    application.register("dataSharing", DataSharing);
    application.register("search", Search);
    application.register("naming", Naming);
    application.register("likes", Reputation);

    initCollapsible();
};

function initCollapsible() {
    // https://www.w3schools.com/howto/howto_js_collapsible.asp
    var coll = document.getElementsByClassName("collapsible");
    for (var i = 0; i < coll.length; i++) {
        coll[i].addEventListener("click", function () {
            var content = this.nextElementSibling;
            if (this.classList.contains("active")) {
                content.style.display = "none";
                this.classList.remove("active");
            } else {
                content.style.display = "block";
                this.classList.add("active");
            }
        });

        var content = coll[i].nextElementSibling;
        if (coll[i].classList.contains("active")) {
            content.style.display = "block";
        } else {
            content.style.display = "none";
        }
    }
}

// BaseElement is inherited by all the controllers. It provides common methods.
class BaseElement extends Stimulus.Controller {

    // checkInputs is a utility function to checks if form inputs are empty or
    // not. It takes care of displaying an appropriate message if a validation
    // fails. The caller should exit if this function returns false.
    checkInputs(...els) {
        for (let i = 0; i < els.length; i++) {
            const val = els[i].value;
            if (val == "" || val == undefined) {
                this.flash.printError(`form validation failed: "${els[i].name}" is empty or invalid`);
                return false;
            }
        }

        return true;
    }

    // fetch fetches the addr and checks for a 200 status in return.
    async fetch(addr, opts) {
        const resp = await fetch(addr, opts);
        if (resp.status != 200) {
            const text = await resp.text();
            throw `wrong status: ${resp.status} ${resp.statusText} - ${text}`;
        }

        return resp;
    }

    get flash() {
        const element = document.getElementById("flash");
        return this.application.getControllerForElementAndIdentifier(element, "flash");
    }

    get peerInfo() {
        const element = document.getElementById("peerInfo");
        return this.application.getControllerForElementAndIdentifier(element, "peerInfo");
    }
}

class Flash extends Stimulus.Controller {
    static get targets() {
        return ["wrapper"];
    }

    printError(e) {
        const flash = this.createFlash("error", e);
        this.wrapperTarget.appendChild(flash);
    }

    printSuccess(e) {
        const flash = this.createFlash("success", e);
        this.wrapperTarget.appendChild(flash);
    }

    createFlash(className, content) {
        content = content.replace(/\s+/g, ' ').trim();

        const newDiv = document.createElement("div");
        newDiv.classList.add("flash-msg", className);

        const closeBtn = document.createElement("span");

        closeBtn.innerHTML = "&times;";
        closeBtn.classList.add("closebtn");
        closeBtn.addEventListener("click", (e) => {
            e.target.parentElement.remove();
        });

        const progress = document.createElement("span");
        progress.classList.add("progress");

        newDiv.innerText = content;
        newDiv.appendChild(closeBtn);
        newDiv.appendChild(progress);

        setTimeout(() => {
            newDiv.remove();
        }, 4000);

        return newDiv;
    }
}

class PeerInfo extends Stimulus.Controller {
    static get targets() {
        return ["name", "email", "phone", "peerAddr", "socketAddr"];
    }

    async initialize() {
        const queryDict = this.getQueryArgs();
        const name = decodeURIComponent(queryDict["name"]).replace("+", " ");
        const email = decodeURIComponent(queryDict["email"]);
        const phone = decodeURIComponent(queryDict["phone"]);
        const endpoint = "http://" + decodeURIComponent(queryDict["addr"]);
        this.name = name;
        this.email = email;
        this.phone = phone;
        this.endpoint = endpoint;
        this.nameTarget.innerText = this.name;
        this.emailTarget.innerText = this.email;
        this.phoneTarget.innerText = this.phone;
        this.peerAddrTarget.innerText = this.endpoint;

        const addr = this.endpoint + "/socket/address";

        try {
            const resp = await fetch(addr);
            const text = await resp.text();

            this.socketAddrTarget.innerText = "tcp://" + text;
            this.socketAddr = text;
        } catch (e) {
            this.flash.printError("failed to fetch socket address: " + e);
        }
    }

    getAPIURL(suffix) {
        return this.endpoint + suffix;
    }

    get flash() {
        const element = document.getElementById("flash");
        return this.application.getControllerForElementAndIdentifier(element, "flash");
    }

    // getQueryArgs returns a dictionary of key=value arguments from the URL
    getQueryArgs() {
        const queryDict = {};
        location.search.substr(1).split("&").forEach(function (item) {
            queryDict[item.split("=")[0]] = item.split("=")[1];
        });

        return queryDict;
    }
}

class Messaging extends BaseElement {
    static get targets() {
        return ["holder", "messages"];
    }

    addMsg(el) {
        this.messagesTarget.append(el);
        this.holderTarget.scrollTop = this.holderTarget.scrollHeight;
    }
}

class Reputation extends BaseElement {

    // reputation 
    async updateMsgReputation(isLike, msgID, msgSenderID, likerID, score) {
        const proxyAddressElement = document.querySelector('td[data-peerinfo-target="peerAddr"]');
        const proxyAddressText = proxyAddressElement.textContent;
        const typeMsg = (isLike) ? "like" : "dislike";
        const addr = proxyAddressText + "/messaging/" + typeMsg;

        const msg = {
            "likerID": likerID,
            "msgSenderID": msgSenderID,
            "messID": msgID,
            "score": score
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(msg)
        };
        try {
            await this.fetch(addr, fetchArgs);
        } catch (e) {
        }
    }

}

class Unicast extends BaseElement {
    static get targets() {
        return ["message", "destination"];
    }

    async send() {
        const addr = this.peerInfo.getAPIURL("/messaging/unicast");

        const ok = this.checkInputs(this.messageTarget, this.destinationTarget);
        if (!ok) {
            return;
        }

        const message = this.messageTarget.value;
        const destination = this.destinationTarget.value;

        const pkt = {
            "Dest": destination,
            "Msg": {
                "Type": "chat",
                "payload": {
                    "Message": message
                }
            }
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(pkt)
        };

        try {
            await this.fetch(addr, fetchArgs);

            const date = new Date();
            const el = document.createElement("div");

            el.classList.add("sent");
            el.innerHTML = `
                <div>
                    <p class="msg">${message}</p>
                    <p class="details">
                        to ${destination} 
                        at ${date.getHours()}:${date.getMinutes()}:${date.getSeconds()}
                    </p>
                </div>`;

            this.messagingController.addMsg(el);
        } catch (e) {
            this.flash.printError("failed to send message: " + e);
        }
    }

    get messagingController() {
        const element = document.getElementById("messaging");
        return this.application.getControllerForElementAndIdentifier(element, "messaging");
    }
}

class Routing extends BaseElement {
    static get targets() {
        return ["table", "graphviz", "peer", "origin", "relay"];
    }

    initialize() {
        this.update();
        this.initIdentityCheck();
    }

    async update() {
        const addr = this.peerInfo.getAPIURL("/messaging/routing");

        try {
            const resp = await this.fetch(addr);
            const data = await resp.json();

            this.tableTarget.innerHTML = "";

            for (const [origin, relay] of Object.entries(data)) {
                const el = document.createElement("tr");

                el.innerHTML = `<td>${origin}</td><td>${relay}</td>`;
                this.tableTarget.appendChild(el);
            }

            this.flash.printSuccess("Routing table updated");

        } catch (e) {
            this.flash.printError("Failed to fetch routing: " + e);
        }

        const graphAddr = addr + "?graphviz=on";

        try {
            const resp = await this.fetch(graphAddr);
            const data = await resp.text();

            var viz = new Viz();

            const element = await viz.renderSVGElement(data);
            this.graphvizTarget.innerHTML = "";
            this.graphvizTarget.appendChild(element);
        } catch (e) {
            this.flash.printError("Failed to display routing: " + e);
        }
    }

    async initIdentityCheck() {
        const addr = this.peerInfo.getAPIURL("/identity/check");

        const info = {
            "Name": this.peerInfo.name,
            "Email": this.peerInfo.email,
            "Phone": this.peerInfo.phone
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(info)
        };

        try {
            await this.fetch(addr, fetchArgs);

            this.flash.printSuccess("Welcome " + this.peerInfo.name + "! Your identity will be verified soon.");

        } catch (e) {
            this.flash.printError("failed to send identity check: " + e);
        }
    }

    async addPeer() {
        const ok = this.checkInputs(this.peerTarget);
        if (!ok) {
            return;
        }

        const addr = this.peerInfo.getAPIURL("/messaging/peers");
        const peer = this.peerTarget.value;

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify([peer])
        };

        try {
            await this.fetch(addr, fetchArgs);
            this.flash.printSuccess("peer added");
            this.update();
        } catch (e) {
            this.flash.printError("failed to add peer: " + e);
        }
    }

    async setEntry() {
        const addr = this.peerInfo.getAPIURL("/messaging/routing");

        const ok = this.checkInputs(this.originTarget);
        if (!ok) {
            return;
        }

        const origin = this.originTarget.value;
        const relay = this.relayTarget.value;

        const message = {
            "Origin": origin,
            "RelayAddr": relay
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(message)
        };

        try {
            await this.fetch(addr, fetchArgs);

            if (relay == "") {
                this.flash.printSuccess("Entry deleted");
            } else {
                this.flash.printSuccess("Entry set");
            }

            this.update();

        } catch (e) {
            this.flash.printError("failed to set entry: " + e);
        }
    }
}

var nbMsg = 0;
class Packets extends BaseElement {
    static get targets() {
        return ["follow", "holder", "scroll", "packets"];
    }

    initialize() {
        const addr = this.peerInfo.getAPIURL("/registry/pktnotify");
        const newPackets = new EventSource(addr);

        newPackets.onmessage = this.packetMessage.bind(this);
        newPackets.onerror = this.packetError.bind(this);

        this.holderTarget.addEventListener("scroll", this.packetsScroll.bind(this));
    }

    packetMessage(e) {
        const pkt = JSON.parse(e.data);
        if (pkt.Msg.Type == "chat") {
            const date = new Date(pkt.Header.Timestamp / 1000000);

            const el = document.createElement("div");

            if (pkt.Header.Source == this.peerInfo.socketAddr) {
                el.classList.add("sent");
            } else {
                el.classList.add("received");
            }

            // server
            // let ws;

            // function init() {
            //     if (ws) {
            //         ws.onerror = ws.onopen = ws.onclose = null;
            //         ws.close();
            //     }

            //     ws = new WebSocket('ws://localhost:6969');
            //     ws.onopen = () => {
            //         console.log('Connection opened!');
            //     }
            //     ws.onmessage = (event) => {
            //         event.data.text().then((text) => {
            //             const data = JSON.parse(text);
            //             showLikes(data[0], data[1], data[2], data[3]);
            //         });
            //     }
            //     ws.onclose = function () {
            //         ws = null;
            //     }
            // }

            // init();

            // ------------ end server ------------ 

            function showLikes(nbLikes, nbDisLikes, likeValueID, dislikeValueID, scoreID) {
                document.getElementById(likeValueID).innerText = nbLikes;
                document.getElementById(dislikeValueID).innerText = nbDisLikes;
                document.getElementById(scoreID).innerText = nbLikes - nbDisLikes;
            }
            function handleLikeClick() {
                const likeElem = document.getElementById('val' + likeButt.id)
                let currLikesNb = likeElem.innerText
                currLikesNb = Number(currLikesNb);
                currLikesNb++;
                likeElem.innerText = currLikesNb;
                const dislikeButtID = likeButt.id.replace(/0/, "1")
                const currDisLikesNb = document.getElementById('val' + dislikeButtID).innerText
                const likeDislikes = Array.of(currLikesNb, currDisLikesNb, 'val' + likeButt.id, 'val' + dislikeButtID);
                // ws.send(JSON.stringify(likeDislikes));
                const scoreID = 'score' + el.id;
                showLikes(likeDislikes[0], likeDislikes[1], likeDislikes[2], likeDislikes[3], scoreID);
                likeButt.removeEventListener('click', handleLikeClick);

                // send Like Msg to container id (IP of the node who created the message)
                // use the like button id to identify the message ID 
                const reputation = new Reputation();
                const likerIP = document.querySelector('td[data-peerinfo-target="socketAddr"]').textContent.slice(6);
                reputation.updateMsgReputation(true, el.id, container.id, likerIP, document.getElementById(scoreID).innerText);

            }
            function handleDisLikeClick() {
                const dislikeElem = document.getElementById('val' + dislikeButt.id)
                let currDisLikesNb = dislikeElem.innerText
                currDisLikesNb = Number(currDisLikesNb)
                currDisLikesNb++
                dislikeElem.innerText = currDisLikesNb;
                const likeButtID = dislikeButt.id.replace(/1/, "0")
                const currLikesNb = document.getElementById('val' + likeButtID).innerText
                const likeDislikes = Array.of(currLikesNb, currDisLikesNb, 'val' + likeButtID, 'val' + dislikeButt.id);
                // ws.send(JSON.stringify(likeDislikes));
                const scoreID = 'score' + el.id;
                showLikes(likeDislikes[0], likeDislikes[1], likeDislikes[2], likeDislikes[3], scoreID);
                dislikeButt.removeEventListener('click', handleDisLikeClick);

                // send DisLike Msg to container id (IP of the node who created the message)
                const reputation = new Reputation();
                const likerIP = document.querySelector('td[data-peerinfo-target="socketAddr"]').textContent.slice(6);
                reputation.updateMsgReputation(false, el.id, container.id, likerIP, document.getElementById(scoreID).innerText);

            }

            const container = document.createElement('div');
            container.style.display = 'flex';  // set the display property to flex
            container.style.justifyContent = 'space-between';  // distribute the items evenly along the main axis

            const size = '20px'
            const likeButt = document.createElement('button');
            likeButt.id = '0' + nbMsg.toString();
            likeButt.style.height = size;
            likeButt.style.width = size;
            likeButt.style.display = 'flex';  // set the display property to flex
            likeButt.style.alignItems = 'center';  // center the items vertically
            likeButt.style.justifyContent = 'center';  // center the items horizontally
            likeButt.style.value
            likeButt.addEventListener('click', handleLikeClick);
            const likeImg = document.createElement('img');
            likeImg.src = './icons8-red-heart-96.png';
            likeImg.style.height = size;
            likeImg.style.width = size;
            likeButt.appendChild(likeImg);

            const dislikeButt = document.createElement('button');
            dislikeButt.id = '1' + nbMsg.toString();
            dislikeButt.style.height = size;
            dislikeButt.style.width = size;
            dislikeButt.style.display = 'flex';  // set the display property to flex
            dislikeButt.style.alignItems = 'center';  // center the items vertically
            dislikeButt.style.justifyContent = 'center';  // center the items horizontally
            dislikeButt.addEventListener('click', handleDisLikeClick);
            const dislikeImg = document.createElement('img');
            dislikeImg.style.height = size;
            dislikeImg.style.width = size;
            dislikeImg.src = './icons8-broken-heart-96.png';
            dislikeButt.appendChild(dislikeImg);
            dislikeButt.style.marginLeft = '15px';

            const likeValue = document.createElement('span');
            likeValue.id = 'val0' + nbMsg.toString();
            likeValue.style.marginRight = '15px';

            const dislikeValue = document.createElement('span');
            dislikeValue.id = 'val1' + nbMsg.toString();

            el.id = pkt.Header.Timestamp;
            const score = document.createElement('span');
            score.id = 'score' + el.id;

            container.appendChild(likeButt);
            container.appendChild(likeValue);
            container.appendChild(score);
            container.appendChild(dislikeButt);
            container.appendChild(dislikeValue);

            // note that this is not secure and prone to XSS attack.
            // displays only the last 5 digits of the IP address for better readability.
            el.innerHTML = `<div><p class="msg">${pkt.Msg.Payload.Message}</p><p class="details">from 
            <span class="ip-addr"> ${pkt.Header.Source.slice(-5)} </span> at ${date.getHours()}:${date.getMinutes()}:${date.getSeconds()}</p></div>`;

            el.appendChild(container);
            this.messagingController.addMsg(el);
            nbMsg++;
            // make the container (message) belongs to the node sending the message 
            // using the ip of the pkt source 
            container.id = pkt.Header.Source;
        }

        const el = document.createElement("div");
        el.innerHTML = `<pre>${JSON.stringify(pkt, null, 2)}</pre>`;
        this.packetsTarget.append(el);

        this.scrollTarget.style.width = this.packetsTarget.scrollWidth + "px";

        if (this.followTarget.checked) {
            this.packetsTarget.scrollLeft = this.packetsTarget.scrollWidth;
            this.holderTarget.scrollLeft = this.holderTarget.scrollWidth;
        }
    }

    packetError() {
        this.flash.printError("failed to listen pkt: stopped listening");
    }

    packetsScroll() {
        this.packetsTarget.scrollLeft = this.holderTarget.scrollLeft;
    }

    get messagingController() {
        return this.application.getControllerForElementAndIdentifier(document.getElementById("messaging"), "messaging");
    }
}



class Broadcast extends BaseElement {
    static get targets() {
        return ["chatMessage", "privateMessage", "privateRecipients"];
    }

    async sendChat() {
        const addr = this.peerInfo.getAPIURL("/messaging/broadcast");

        const ok = this.checkInputs(this.chatMessageTarget);
        if (!ok) {
            return;
        }

        const message = this.chatMessageTarget.value;

        const msg = {
            "Type": "chat",
            "payload": {
                "Message": message
            }
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(msg)
        };

        try {
            await this.fetch(addr, fetchArgs);
            this.flash.printSuccess("chat message broadcasted");
        } catch (e) {
            this.flash.printError("failed to send message: " + e);
        }
    }

    async sendPrivate() {
        const addr = this.peerInfo.getAPIURL("/messaging/broadcast");

        const ok = this.checkInputs(this.privateMessageTarget, this.privateRecipientsTarget);
        if (!ok) {
            return;
        }

        const destination = this.privateRecipientsTarget.value;
        const message = this.privateMessageTarget.value;

        const recipients = {};
        destination.split(",").forEach(e => recipients[e.trim()] = {});

        const msg = {
            "Type": "private",
            "payload": {
                "Recipients": recipients,
                "Msg": {
                    "Type": "chat",
                    "payload": {
                        "Message": message
                    }
                }
            }
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(msg)
        };

        try {
            await this.fetch(addr, fetchArgs);
            this.flash.printSuccess("private message sent");
        } catch (e) {
            this.flash.printError("failed to send message: " + e);
        }
    }
}

class Catalog extends BaseElement {
    static get targets() {
        return ["content", "key", "value"];
    }

    initialize() {
        this.update();
    }

    async update() {
        const addr = this.peerInfo.getAPIURL("/datasharing/catalog");

        try {
            const resp = await this.fetch(addr);
            const data = await resp.json();

            this.contentTarget.innerHTML = "";

            // Expected format of data:
            //
            // {
            //     "chunk1": {
            //         "peerA": {}, "peerB": {},
            //     },
            //     "chunk2": {...},
            // }

            if (Object.keys(data).length === 0) {
                this.contentTarget.innerHTML = "<i>no elements</i>";
                this.flash.printSuccess("Catalog updated, nothing found");
                return;
            }

            for (const [chunk, peersBag] of Object.entries(data)) {
                const entry = document.createElement("div");
                const chunkName = document.createElement("p");
                chunkName.innerHTML = chunk;

                entry.appendChild(chunkName);

                for (var peer in peersBag) {
                    const peerEl = document.createElement("p");
                    peerEl.innerHTML = peer;
                    entry.appendChild(peerEl);
                }

                this.contentTarget.appendChild(entry);
            }

            this.flash.printSuccess("Catalog updated");
        } catch (e) {
            this.flash.printError("Failed to fetch catalog: " + e);
        }
    }

    async add() {
        const addr = this.peerInfo.getAPIURL("/datasharing/catalog");

        const ok = this.checkInputs(this.keyTarget, this.valueTarget);
        if (!ok) {
            return;
        }

        const key = this.keyTarget.value;
        const value = this.valueTarget.value;

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify([key, value])
        };

        try {
            await this.fetch(addr, fetchArgs);
            this.update();
        } catch (e) {
            this.flash.printError("failed to add catalog entry: " + e);
        }
    }
}

class DataSharing extends BaseElement {
    static get targets() {
        return ["uploadResult", "fileUpload", "downloadMetahash"];
    }

    async upload() {
        const addr = this.peerInfo.getAPIURL("/datasharing/upload");

        const fileList = this.fileUploadTarget.files;

        if (fileList.length == 0) {
            this.flash.printError("No file found");
            return;
        }

        const file = fileList[0];

        const reader = new FileReader();
        reader.addEventListener('load', async (event) => {
            const result = event.target.result;

            const fetchArgs = {
                method: "POST",
                headers: {
                    "Content-Type": "multipart/form-data"
                },
                body: result
            };

            try {
                const resp = await this.fetch(addr, fetchArgs);
                const text = await resp.text();

                this.flash.printSuccess(`data uploaded, metahash: ${text}`);
                this.uploadResultTarget.innerHTML = `Metahash: ${text}`;
            } catch (e) {
                this.flash.printError("failed to upload data: " + e);
            }
        });

        reader.addEventListener('progress', (event) => {
            if (event.loaded && event.total) {
                const percent = (event.loaded / event.total) * 100;
                this.flash.printSuccess(`File upload progress: ${Math.round(percent)}`);
            }
        });
        reader.readAsArrayBuffer(file);
    }

    async download() {
        const ok = this.checkInputs(this.downloadMetahashTarget);
        if (!ok) {
            return;
        }

        const metahash = this.downloadMetahashTarget.value;

        const addr = this.peerInfo.getAPIURL("/datasharing/download?key=" + metahash);

        try {
            const resp = await this.fetch(addr);
            const blob = await resp.blob();

            this.triggerDownload(metahash, blob);
            this.flash.printSuccess("Data downloaded!");
        } catch (e) {
            this.flash.printError("Failed to download data: " + e);
        }
    }

    triggerDownload(metahash, blob) {
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement("a");

        a.style.display = "none";
        a.href = url;
        a.download = metahash;
        document.body.appendChild(a);

        a.click();
        window.URL.revokeObjectURL(url);
    }
}

class Search extends BaseElement {
    static get targets() {
        return ["searchAllResult", "searchAllPattern", "searchAllBudget", "searchAllTimeout",
            "searchFirstResult", "searchFirstPattern", "searchFirstInitialBudget",
            "searchFirstFactor", "searchFirstRetry", "searchFirstTimeout"];
    }

    async searchAll() {
        const addr = this.peerInfo.getAPIURL("/datasharing/searchAll");

        const ok = this.checkInputs(this.searchAllPatternTarget,
            this.searchAllBudgetTarget, this.searchAllTimeoutTarget);
        if (!ok) {
            return;
        }

        const pattern = this.searchAllPatternTarget.value;
        const budget = this.searchAllBudgetTarget.value;
        const timeout = this.searchAllTimeoutTarget.value;

        const pkt = {
            "Pattern": pattern,
            "Budget": parseInt(budget),
            "Timeout": timeout
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "multipart/form-data"
            },
            body: JSON.stringify(pkt)
        };

        try {
            this.searchAllResultTarget.innerHTML = `<i>searching all...</i>`;

            const resp = await this.fetch(addr, fetchArgs);
            const text = await resp.text();

            this.flash.printSuccess(`SearchAll done, result: ${text}`);
            this.searchAllResultTarget.innerHTML = `Names: ${text}`;
        } catch (e) {
            this.flash.printError("failed to searchAll: " + e);
        }
    }

    async searchFirst() {
        const addr = this.peerInfo.getAPIURL("/datasharing/searchFirst");

        const ok = this.checkInputs(this.searchFirstPatternTarget, this.searchFirstInitialBudgetTarget,
            this.searchFirstFactorTarget, this.searchFirstRetryTarget, this.searchFirstTimeoutTarget);
        if (!ok) {
            return;
        }

        const pattern = this.searchFirstPatternTarget.value;
        const initial = this.searchFirstInitialBudgetTarget.value;
        const factor = this.searchFirstFactorTarget.value;
        const retry = this.searchFirstRetryTarget.value;
        const timeout = this.searchFirstTimeoutTarget.value;

        const pkt = {
            "Pattern": pattern,
            "Initial": parseInt(initial),
            "Factor": parseInt(factor),
            "Retry": parseInt(retry),
            "Timeout": timeout
        };

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "multipart/form-data"
            },
            body: JSON.stringify(pkt)
        };

        try {
            this.searchFirstResultTarget.innerHTML = `<i>searching first...</i>`;

            const resp = await this.fetch(addr, fetchArgs);
            const text = await resp.text();

            this.flash.printSuccess(`Search done, result: ${text}`);
            if (text == "") {
                this.searchFirstResultTarget.innerHTML = "<i>Nothing found</i>";
            } else {
                this.searchFirstResultTarget.innerHTML = `Names: ${text}`;
            }

            this.flash.printSuccess(`search first done, result: ${text}`);
        } catch (e) {
            this.flash.printError("failed to Search first: " + e);
        }
    }
}

class Naming extends BaseElement {
    static get targets() {
        return ["resolveResult", "resolveFilename", "tagFilename", "tagMetahash"];
    }

    async resolve() {
        this.checkInputs(this.resolveFilenameTarget);

        const filename = this.resolveFilenameTarget.value;

        const addr = this.peerInfo.getAPIURL("/datasharing/naming?name=" + filename);

        try {
            const resp = await this.fetch(addr);
            const text = await resp.text();

            if (text == "") {
                this.resolveResultTarget.innerHTML = "<i>nothing found</i>";
            } else {
                this.resolveResultTarget.innerHTML = text;
            }

            this.flash.printSuccess("filename resolved");
        } catch (e) {
            this.flash.printError("Failed to resolve filename: " + e);
        }
    }

    async tag() {
        const addr = this.peerInfo.getAPIURL("/datasharing/naming");

        const ok = this.checkInputs(this.tagFilenameTarget, this.tagMetahashTarget);
        if (!ok) {
            return;
        }

        const filename = this.tagFilenameTarget.value;
        const metahash = this.tagMetahashTarget.value;

        const fetchArgs = {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify([filename, metahash])
        };

        try {
            await this.fetch(addr, fetchArgs);
            this.flash.printSuccess("tagging done");
        } catch (e) {
            this.flash.printError("failed to tag filename: " + e);
        }
    }
}
