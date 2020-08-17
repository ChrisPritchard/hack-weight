
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

document.querySelector("#show-set-goals").addEventListener("click", function() {
    showGoalsSection();
});

document.querySelector("#show-add-calorie-entry").addEventListener("click", function() {
    showAddEntrySection();
});

var cancelElems = document.querySelectorAll(".cancel-button");
for (var i = 0; i < cancelElems.length; i++) {
    cancelElems[i].addEventListener("click", function() {
        changeSection("#today-section");
    });
}

document.querySelector("#set-weight").addEventListener("click", function() {
    var weight = document.querySelector("#weight-to-set").value;
    sendData("/today/weight", "weight="+weight, function() {
        showTodaySection();
    });
});

var goalsElems = {
    currentWeight: document.querySelector("#current-weight"),
    targetWeight: document.querySelector("#target-weight"),
    targetDate: document.querySelector("#target-date"),
    dailyBurnRate: document.querySelector("#daily-burn-rate"),
};
goalsElems.currentWeight.addEventListener("change", function() { calculateRates(); });
goalsElems.targetWeight.addEventListener("change", function() { calculateRates(); });
goalsElems.targetDate.addEventListener("change", function() { calculateRates(); });
goalsElems.dailyBurnRate.addEventListener("change", function() { calculateRates(); });

document.querySelector("#clear-history").addEventListener("click", function() {
    if(confirm("Are you sure? This is irreversable")) {
        sendData("/history/clear", null, function() {
            goalsElems.currentWeight.value = "";
            goalsElems.targetWeight.value = "";
            goalsElems.targetDate.value = "";
            goalsElems.dailyBurnRate.value = "";
            document.querySelector("#weight-to-set").value = "";
            showTodaySection();
        });
    }
});

function calculateRates() {
    document.querySelector("#goals-description").value = "";
    document.querySelector("#set-goals").setAttribute("disabled", "disabled");

    if (!goalsElems.dailyBurnRate.value) {
        return;
    }
    var days = (Date.parse(goalsElems.targetDate.value) - (new Date()).getTime()) / (1000 * 3600 * 24);
    if(isNaN(days))
        return;

    var calsPerKG = 7700;
    var toLose = goalsElems.currentWeight.value - goalsElems.targetWeight.value;
    var deficitPerDay = (toLose * calsPerKG) / days;
    var target = goalsElems.dailyBurnRate.value - deficitPerDay;
    if (isNaN(target) || target < 500)
        return;

    document.querySelector("#goals-description").innerText = Math.round(target)+" calories per day to meet goal";
    document.querySelector("#set-goals").removeAttribute("disabled");
}

document.querySelector("#set-goals").addEventListener("click", function() {
    var data = "target_weight=" + goalsElems.targetWeight.value;
    data += "&target_date=" + goalsElems.targetDate.value;
    data += "&daily_burn_rate=" + goalsElems.dailyBurnRate.value;
    sendData("/goals", data, function() {
        showTodaySection();
    });
});

document.querySelector("#add-entry").addEventListener("click", function() {
    var amount = document.querySelector("#amount-to-set").value;
    var category = document.querySelector("#new-category-to-set").value;
    if (category == "") {
        category = document.querySelector("#existing-category-to-set").value;
        if (category == "(Select)")
            category = "";
    }
    var existingCategory = document.querySelector("option[value=\""+category+"\"]");
    if (!existingCategory) {
        document.querySelector("#existing-category-to-set").innerHTML += "<option value=\""+category+"\">"+category+"</option>";
    }
    sendData("/today/calories", "amount="+amount+"&category="+category, function() {
        showTodaySection();
    });
});

function showTodaySection() {
    today = ["Sunday","Monday","Tuesday","Wednesday","Thursday","Friday","Saturday"][new Date().getDay()];
    document.querySelector("#today").innerText = today;

    getResponse("/today", function(today) {
        if (today.Weight && today.Weight != 0) {
            weightText = (Math.round(today.Weight * 10)/10).toFixed(1)
            document.querySelector("#recorded-weight").innerText = "Today's Weight: "+weightText+" KG";
            document.querySelector("#current-weight").value = today.Weight;
            document.querySelector("#weight-to-set").value = today.Weight;
        } else {
            document.querySelector("#show-set-weight").classList.remove("hide");
        }
    
        if (today.LastWeight && today.LastWeight != 0) {
            document.querySelector("#weight-to-set").value = today.LastWeight;
            document.querySelector("#current-weight").value = today.LastWeight;
        }

        var totalConsumed = 0;
        var entries = document.querySelector("#today-entries");
        entries.innerHTML = "";
        for(var i = 0; i < today.Calories.length; i++) {
            var entry = today.Calories[i];
            totalConsumed += entry.Amount;
            var htmlToAdd = "<tr><td>"+entry.Amount+" Cal</td>"
            htmlToAdd += "<td>"+entry.Category+"</td>"
            htmlToAdd += "<td><button data-delete=\""+entry.ID+"\">X</button></td></tr>"
            entries.innerHTML += htmlToAdd;
        }
        document.querySelector("#total-consumed").innerText = "Consumed Today: "+totalConsumed+" Cal";

        var deletables = document.querySelectorAll("[data-delete]");
        for (var i = 0; i < deletables.length; i++) {
            deletables[i].addEventListener("click", function(e) {
                if (!confirm("Are you sure?"))
                    return;
                var id = e.target.getAttribute("data-delete");
                sendData("/calories/delete", "id="+id, function() {
                    showAddEntrySection(true);
                    showTodaySection();
                })
            })
        }

        if (today.TodayMax && today.TodayMax > 0) {
            document.querySelector("#total-consumed").innerText += " / " + today.TodayMax + " Cal";
        }

        changeSection("#today-section");
    });
}

function showAddEntrySection(dontSwitch) {
    getResponse("/categories", function(categories) {
        var select = document.querySelector("#existing-category-to-set");
        select.innerHTML = "<option selected>(Select)</option>";
        for (var i = 0; i < categories.length; i++) {
            select.innerHTML += "<option value=\""+categories[i]+"\">"+categories[i]+"</option>";
        }
        document.querySelector("#amount-to-set").value = 200;
        document.querySelector("#new-category-to-set").value = "";

        if(!dontSwitch)
            changeSection("#add-calorie-entry-section");
    });
}

function showGoalsSection(dontSwitch) {
    getResponse("/goals", function(goals) {
        if (goals.TargetWeight && goals.TargetWeight != 0) {
            document.querySelector("#target-weight").value = goals.TargetWeight;
        }
    
        if (goals.TargetDate) {
            document.querySelector("#target-date").value = goals.TargetDate;
        }

        if (goals.BurnRate && goals.BurnRate != 0) {
            document.querySelector("#daily-burn-rate").value = goals.BurnRate;
        }

        if(!dontSwitch)
            changeSection("#set-goals-section");
    });
}

showAddEntrySection(false);
showGoalsSection(false);
showTodaySection();