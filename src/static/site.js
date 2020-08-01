
function getResponse(path, onResult) {
    var request = new XMLHttpRequest();
    request.open('GET', '/categories', true);
    request.setRequestHeader("Content-Type", "application/json");
    request.onload = function() {
        var resp = this.response;
        onResult(JSON.parse(resp));
    };
    request.send();
}

getResponse("/today", function(today) {
    if (today.Weight && today.Weight != 0) {
        document.querySelector("#recorded-weight").innerText == "Today's Weight: "+today.Weight + " KG";
    } else {
        document.querySelector("#set-weight-button").classList.remove("hide");
    }

    var totalConsumed = 0;
    var entries = document.querySelector("#today-entries");
    for(var i = 0; i < today.Calories; i++) {
        var entry = today.Calories[i];
        totalConsumed += entry.Amount;
        entries.innerHTML += "<li>"+entry.Amount+" Cal - "+entry.Category+"</li>";
    }
    document.querySelector("#total-consumed").innerText = "Total Consumed Today: "+totalConsumed+" Cal";
});

getResponse("/categories", function(categories) {
    var select = document.querySelector("#categories");
    for (var i = 0; i < categories.length; i++) {
        select.innerHTML += "<option>"+categories[i]+"</option>";
    }
});

function showTodaySection() {
    today = ["Sunday","Monday","Tuesday","Wednesday","Thursday","Friday","Saturday"][new Date().getDay()];
    document.querySelector("#today").innerText = today;
}

showTodaySection();