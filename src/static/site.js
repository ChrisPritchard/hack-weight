
function getResponse(path, onResult) {
    var request = new XMLHttpRequest();
    request.open('GET', path, true);
    request.setRequestHeader("Content-Type", "application/json");
    request.onload = function() {
        var resp = this.response;
        onResult(JSON.parse(resp));
    };
    request.send();
}

function sendData(path, body, onSuccess) {
    var request = new XMLHttpRequest();
    request.open("POST", path, true);
    request.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
    request.onreadystatechange = function() { // Call a function when the state changes.
        if (this.readyState === XMLHttpRequest.DONE && this.status === 202) {
            onSuccess();
        }
    }
    request.send(body);
}

function changeSection(newSectionSelector) {
    var sections = document.querySelectorAll(".section");
    for(var i = 0; i < sections.length; i++) {
        sections[i].classList.add('hide');
    }
    document.querySelector(newSectionSelector).classList.remove("hide");
}

document.querySelector("#show-set-weight").addEventListener("click", function() {
    changeSection("#set-weight-section");
});

document.querySelector("#show-add-calorie-entry").addEventListener("click", function() {
    showAddEntrySection();
});

document.querySelector("#set-weight").addEventListener("click", function() {
    var weight = document.querySelector("#weight-to-set").value;
    sendData("/today/weight", "weight="+weight, function() {
        showTodaySection();
    });
});

function showTodaySection() {
    today = ["Sunday","Monday","Tuesday","Wednesday","Thursday","Friday","Saturday"][new Date().getDay()];
    document.querySelector("#today").innerText = today;

    getResponse("/today", function(today) {
        if (today.Weight && today.Weight != 0) {
            document.querySelector("#recorded-weight").innerText = "Today's Weight: "+today.Weight + " KG";
        } else {
            document.querySelector("#show-set-weight").classList.remove("hide");
        }
    
        var totalConsumed = 0;
        var entries = document.querySelector("#today-entries");
        for(var i = 0; i < today.Calories.length; i++) {
            var entry = today.Calories[i];
            totalConsumed += entry.Amount;
            entries.innerHTML += "<li>"+entry.Amount+" Cal - "+entry.Category+"</li>";
        }
        document.querySelector("#total-consumed").innerText = "Total Consumed Today: "+totalConsumed+" Cal";

        changeSection("#today-section");
    });
}

function showAddEntrySection() {
    getResponse("/categories", function(categories) {
        var select = document.querySelector("#existing-category-to-set");
        select.innerHTML = "<option selected>(Select)</option>";
        for (var i = 0; i < categories.length; i++) {
            select.innerHTML += "<option>"+categories[i]+"</option>";
        }
        document.querySelector("#amount-to-set").value = 200;
        document.querySelector("#new-category-to-set").value = "";

        changeSection("#add-calorie-entry-section");
    });
}

document.querySelector("#add-entry").addEventListener("click", function() {
    var amount = document.querySelector("#amount-to-set").value;
    var category = document.querySelector("#new-category-to-set").value;
    if (category == "") {
        category = document.querySelector("#existing-category-to-set").value;
        if (category == "(Select)")
            category = "";
    }
    sendData("/today/calories", "amount="+amount+"&category="+category, function() {
        showTodaySection();
    });
});

showTodaySection();