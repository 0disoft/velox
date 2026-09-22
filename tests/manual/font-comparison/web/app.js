(function () {
var initial = "한글 비교 문장이다. 맑은 고딕과 현재 폰트의 자간과 행간을 본다.";
var second = "English sample: The quick brown fox jumps over 1234567890.";
var third = "숫자 0123456789, 구두점 !?.,;:()[]-/ mixed 가나다 123 ABC";
var text = initial + "\n" + second + "\n" + third;
var left = document.getElementById("left");
var right = document.getElementById("right");
var noto = document.getElementById("noto");
var status = document.getElementById("noto-status");
var areas = [left, right, noto];
for (var i = 0; i < areas.length; i++) { areas[i].value = text; }
var syncing = false;
function syncFrom(from) {
if (syncing) { return; }
syncing = true;
for (var j = 0; j < areas.length; j++) { if (areas[j] !== from) { areas[j].value = from.value; } }
syncing = false;
}
for (var k = 0; k < areas.length; k++) {
(function (area) { area.addEventListener("input", function () { syncFrom(area); }); })(areas[k]);
}
document.getElementById("dpr").textContent = String(window.devicePixelRatio || "-");
function fail() { status.textContent = "실패"; }
try {
if (!("fonts" in document) || typeof document.fonts.load !== "function") { fail(); }
else {
document.fonts.load('400 16px "Velox Noto Sans KR"', "한글 English").then(function (faces) {
if (faces && faces.length > 0) { status.textContent = "성공"; noto.disabled = false; }
else { fail(); }
}).catch(function () { fail(); });
}
} catch (e) { fail(); }
})();
