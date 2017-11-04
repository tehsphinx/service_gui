let log = {
    init: function() {
        // Wait for astilectron to be ready
        document.addEventListener('astilectron-ready', function() {
            // Listen
            log.listen();
            log.startLogging();
        });
    },
    listen: function() {
        astilectron.listen(function(message) {
            switch (message.name) {
                case "log.all":
                    document.getElementById("log").innerHTML = '<pre>' + message.payload + '</pre>';
            }
        });
    },
    startLogging: function() {
        astilectron.send({"name": "log.start"})
    }
};