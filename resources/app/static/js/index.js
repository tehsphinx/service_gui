let index = {
    init: function() {
        // Wait for astilectron to be ready
        document.addEventListener('astilectron-ready', function() {
            // Listen
            index.listen();

            document.querySelector('#btnRestart').addEventListener('click', index.restart);
            document.querySelector('#btnShowLog').addEventListener('click', index.showLog);
        });
    },
    listen: function() {
        astilectron.listen(function(message) {
            switch (message.name) {
                case "set.style":
                    index.listenSetStyle(message);
                    break;
                case "set.running":
                    index.setRunningState(message);
                    break;
            }
        });
    },
    listenSetStyle: function(message) {
        document.body.className = message.payload;
    },
    setRunningState: function(message) {
        let t = (message.payload ? "OK" : "STOPPED");

        document.getElementById("runningState").innerHTML = t;
        document.getElementById("runningState").className = t.toLowerCase();
    },
    restart: function() {
        astilectron.send({"name": "la.restart"})
    },
    showLog: function () {
        astilectron.send({"name": "log.show"})
    }
    // refreshList: function() {
    //     astilectron.send({"name": "get.list"}, function(message) {
    //         if (message.payload.length === 0) {
    //             return
    //         }
    //         let c = `<ul>`;
    //         for (let i = 0; i < message.payload.length; i++) {
    //             c += `<li class="` + message.payload[i].type + `">` + message.payload[i].name + `</li>`
    //         }
    //         c += `</ul>`;
    //         document.getElementById("list").innerHTML = c
    //     })
    // }
};