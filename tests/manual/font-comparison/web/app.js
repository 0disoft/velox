(function () {
var initial = "한글 비교 문장이다. 맑은 고딕과 현재 폰트의 자간과 행간을 본다.";
var second = "English sample: The quick brown fox jumps over 1234567890.";
var third = "숫자 0123456789, 구두점 !?.,;:()[]-/ mixed 가나다 123 ABC";
var text = initial + "\n" + second + "\n" + third;
var left = document.getElementById("left");
var right = document.getElementById("right");
left.value = text;
right.value = text;
var syncing = false;
function sync(from, to) {
if (syncing) { return; }
syncing = true;
to.value = from.value;
syncing = false;
}
left.addEventListener("input", function () { sync(left, right); });
right.addEventListener("input", function () { sync(right, left); });
document.getElementById("dpr").textContent = String(window.devicePixelRatio || "-");
})();
