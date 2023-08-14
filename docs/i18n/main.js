const BASEURL = "https://raw.githubusercontent.com/knadh/listmonk/master/i18n/";
const BASELANG = "en";

var app = new Vue({
	el: "#app",
	data: {
		base: {},
		keys: [],
		visibleKeys: {},
		values: {},
		view: "all",
		loadLang: BASELANG,

		isRawVisible: false,
		rawData: "{}"
	},

	methods: {
		init() {
			document.querySelector("#app").style.display = 'block';
			document.querySelector("#loading").remove();
		},

		loadBaseLang(url) {
			return fetch(url).then(response => response.json()).then(data => {
				// Retain the base values.
				Object.assign(this.base, data);

				// Get the sorted keys from the language map.
				const keys = [];
				const visibleKeys = {};
				let head = null;
				Object.entries(this.base).sort((a, b) => a[0].localeCompare(b[0])).forEach((v) => {
					const h = v[0].split('.')[0];
					keys.push({
						"key": v[0],
						"head": (head !== h ? h : null) // eg: campaigns on `campaigns.something.else`
					});

					visibleKeys[v[0]] = true;
					head = h;
				});

				this.keys = keys;
				this.visibleKeys = visibleKeys;
				this.values = { ...this.base };

				// Is there cached localStorage data?
				if (localStorage.data) {
					try {
						this.populateData(JSON.parse(localStorage.data));
					} catch (e) {
						console.log("Bad JSON in localStorage: " + e.toString());
					}
					return;
				}
			});
		},

		populateData(data) {
			// Filter out all keys from data except for the base ones
			// in the base language.
			const vals = this.keys.reduce((a, key) => {
				a[key.key] = data.hasOwnProperty(key.key) ? data[key.key] : this.base[key.key];
				return a;
			}, {});

			this.values = vals;
			this.saveData();
		},

		loadLanguage(lang) {
			return fetch(BASEURL + lang + ".json").then(response => response.json()).then(data => {
				this.populateData(data);
			}).catch((e) => {
				console.log(e);
				alert("error fetching file: " + e.toString());
			});
		},

		saveData() {
			localStorage.data = JSON.stringify(this.values);
		},

		// Has a key been translated (changed from the base)?
		isDone(key) {
			return this.values[key] && this.base[key] !== this.values[key];
		},

		isItemVisible(key) {
			return this.visibleKeys[key];
		},

		onToggleRaw() {
			if (!this.isRawVisible) {
				this.rawData = JSON.stringify(this.values, Object.keys(this.values).sort(), 4);
			} else {
				try {
					this.populateData(JSON.parse(this.rawData));
				} catch (e) {
					alert("error parsing JSON: " + e.toString());
					return false;
				}
			}

			this.isRawVisible = !this.isRawVisible;
		},

		onLoadLanguage() {
			if (!confirm("Loading this language will overwrite your local changes. Continue?")) {
				return false;
			}

			this.loadLanguage(this.loadLang);
		},

		onNewLang() {
			if (!confirm("Creating a new language will overwrite your local changes. Continue?")) {
				return false;
			}

			let data = { ...this.base };
			data["_.code"] = "iso-code-here"
			data["_.name"] = "New language"
			this.populateData(data);
		},

		onDownloadJSON() {
			// Create a Blob using the content, mimeType, and optional encoding
			const blob = new Blob([JSON.stringify(this.values, Object.keys(this.values).sort(), 4)], { type: "" });

			// Create an anchor element with a download attribute
			const link = document.createElement('a');
			link.download = `${this.values["_.code"]}.json`;
			link.href = URL.createObjectURL(blob);

			// Append the link to the DOM, click it to start the download, and remove it
			document.body.appendChild(link);
			link.click();
			document.body.removeChild(link);
		}
	},

	mounted() {
		this.loadBaseLang(BASEURL + BASELANG + ".json").then(() => this.init());
	},

	watch: {
		view(v) {
			// When the view changes, create a copy of the items to be filtered
			// by and filter the view based on that. Otherwise, the moment the value
			// in the input changes, the list re-renders making items disappear.

			const visibleKeys = {};
			this.keys.forEach((k) => {
				let visible = true;

				if (v === "pending") {
					visible = !this.isDone(k.key);
				} else if (v === "complete") {
					visible = this.isDone(k.key);
				}

				if (visible) {
					visibleKeys[k.key] = true;
				}
			});

			this.visibleKeys = visibleKeys;
		}
	},

	computed: {
		completed() {
			let n = 0;

			this.keys.forEach(k => {
				if (this.values[k.key] !== this.base[k.key]) {
					n++;
				}
			});

			return n;
		}
	}
});
