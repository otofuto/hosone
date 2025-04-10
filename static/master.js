function setHeight() {
    document.documentElement.style.setProperty('--vh', visualViewport.height + 'px');
}
setHeight();
window.onresize = () => setHeight();
window.visualViewport.onresize = () => setHeight();

onload = () => {
    document.querySelectorAll('[data-ot-get]').forEach(elm => {
        if (!elm.getAttribute('data-ot-get').startsWith('~'))
            execGet(elm);
    });
    document.querySelectorAll('form[data-ot-post]').forEach(elm => {
        elm.onsubmit = () => {
            let acts = elm.getAttribute('data-ot-post').split(',');
            if (acts[0] != '') {
                let data = new FormData(elm);
                formDisabled(elm, true);
                post(acts[0], data).then(res => {
                    if (res.result) {
                        if (acts.length > 0) {
                            viewMessage(acts[1]);
                        } else {
                            viewMessage('成功しました');
                        }
                    } else if (res.message && res.message != '') {
                        viewMessage(res.message);
                    } else {
                        viewMessage('失敗しました');
                    }
                }).catch(err => {
                    console.error(err);
                    elm.innerHTML = err;
                }).finally(() => {
                    formDisabled(elm, false);
                    if (typeof getend == 'function') {
                        getend(elm.getAttribute('data-ot-post'));
                    }
                });
            }
            return false;
        }
    });
    let h1 = document.querySelector('body>header>h1');
    if (h1) {
        h1.addEventListener('click', () => location = location.pathname);
    }
};

function execGet(elm, custom, getend) {
    if (elm.getAttribute('data-ot-get') == '') return;
    let url = elm.getAttribute('data-ot-get');
    if (url.startsWith('~')) url = url.substring(1);
    let data = {};
    if (url.indexOf('?') > 0) {
        data = null;
    } else if (elm.getAttribute('data-ot-form') != '') {
        let fm = document.querySelector('form[name="' + elm.getAttribute('data-ot-form') + '"]');
        fm.querySelectorAll('[name]').forEach(inp => {
            data[inp.name] = inp.value;
        });
    } else {
        new URL(location).searchParams.forEach((v, k) => {
            data[k] = v;
        });
    }
    let cnt = 0;
    get(url, data).then(res => {
        if (res.result) {
            if (res.message) {
                setText(elm, res.message);
            } else if (res.html) {
                elm.innerHTML = res.html;
            } else if (Array.isArray(res.list)) {
                let sample = null;
                if (elm.querySelectorAll('.ot-sample').length > 0) {
                    sample = elm.querySelectorAll('.ot-sample');
                    sample.forEach(s => s.classList.remove('ot-sample'));
                    sample.forEach(s => s.removeAttribute('style'));
                } else {
                    sample = [document.createElement('div')];
                }
                cnt = res.list.length;
                res.list.forEach(l => {
                    sample.forEach(sam => {
                        let art = sam.cloneNode(true);
                        for (let i = 0; i < Object.keys(l).length; i++) {
                            art.querySelectorAll('[data-ot-' + Object.keys(l)[i] + ']').forEach(target => {
                                let intext = l[Object.keys(l)[i]];
                                if (typeof custom == 'function') {
                                    intext = custom(Object.keys(l)[i], intext, target);
                                    if (intext == undefined) intext = l[Object.keys(l)[i]];
                                }
                                setText(target, intext);
                            });
                            elm.appendChild(art);
                        }
                    });
                });
                sample.forEach(s => s.classList.add('ot-sample'));
                sample.forEach(s => s.style.display = 'none');
            } else if (res[elm.getAttribute('data-ot-object')]) {
                let l = res[elm.getAttribute('data-ot-object')];
                for (let i = 0; i < Object.keys(l).length; i++) {
                    elm.querySelectorAll('[data-ot-' + Object.keys(l)[i] + ']').forEach(target => {
                        let intext = l[Object.keys(l)[i]];
                        if (typeof custom == 'function') {
                            intext = custom(Object.keys(l)[i], intext, target);
                            if (intext == undefined) intext = l[Object.keys(l)[i]];
                        }
                        setText(target, intext);
                    });
                }
            } else {
                elm.innerText += 'Success';
            }
        } else {
            elm.innerHTML = res.message;
        }
    }).catch(err => {
        console.error(err);
        elm.innerHTML = err;
    }).finally(() => {
        if (typeof getend == 'function') {
            getend(elm.getAttribute('data-ot-get'), cnt);
        }
    });
}

function setText(target, text) {
    let tagname = target.tagName.toLowerCase();
    if (tagname == 'input' || tagname == 'select' || tagname == 'textarea') {
        target.value = text;
    }else if (tagname == 'img') {
        target.src = text;
    } else {
        target.appendChild(document.createTextNode(text));
    }
}

function otClear(target) {
    if (target.querySelector('[ot-sample]')) {
        let sample = target.querySelector('[ot-sample]').cloneNode(true);
        target.innerHTML = '';
        target.appendChild(sample);
    } else if (target.children.length > 0) {
        let sample = target.children[0];
        target.innerHTML = '';
        target.appendChild(sample);
    } else {
        target.innerHTML = '';
    }
}

function getParentByTagName(elm, name) {
    for (let i = 0; elm.parentNode.tagName.toLowerCase() != name.toLowerCase(); i++) {
        elm = elm.parentNode;
        if (i > 100) {
            return null;
        }
    }
    return elm.parentNode;
}

function viewMessage(str, f) {
    let txt = document.createElement('div');
    txt.innerText = str;
    if (document.querySelector('.messagebox')) {
        document.querySelector('.messagebox').appendChild(txt);
    } else {
        let msg = document.createElement('div');
        msg.setAttribute('class', 'messagebox');
        if (f) msg.addEventListener('click', () => {
            msg.remove();
            f();
        });
        else msg.setAttribute('onclick', 'this.remove()');
        msg.appendChild(txt);
        document.body.appendChild(msg);
    }
}

function viewSelection(msg, opts) {
    let back = document.createElement('div');
    back.setAttribute('class', 'selectionback');
    let b = document.createElement('label');
    b.innerHTML = msg.replaceAll('\n', '<br>').replaceAll('\t', '<span class="tab"></span>');
    back.appendChild(b);
    opts.forEach(opt => {
        let a = document.createElement('div');
        a.innerText = opt.text;
        a.addEventListener('click', () => {
            opt.click();
            back.remove();
        });
        back.appendChild(a);
    });
    document.body.appendChild(back);
}

function viewInput(msg, opts) {
    let back = document.createElement('form');
    back.name = 'master-viewinput';
    back.setAttribute('class', 'selectionback');
    let b = document.createElement('label');
    b.innerText = msg;
    back.appendChild(b);
    opts.forEach(opt => {
        if (typeof opt.text == 'string') {
            let a = document.createElement('div');
            a.innerText = opt.text;
            a.addEventListener('click', () => {
                if (opt.submit) {
                    let data = new FormData(back);
                    opt.click(data);
                } else {
                    opt.click();
                }
                back.remove();
            });
            back.appendChild(a);
        } else if (typeof opt.input == 'string') {
            let a = back.children[back.children.length - 1];
            if (a.tagName != 'ARTICLE')
                a = document.createElement('article');
            let lbl = document.createElement('label');
            let spn = document.createElement('span');
            spn.innerText = opt.input;
            lbl.appendChild(spn);
            a.appendChild(lbl);
            let inp = document.createElement('input');
            inp.setAttribute('type', opt.type);
            inp.name = opt.name;
            inp.value = opt.value;
            lbl.appendChild(inp);
            if (typeof opt.datalist == 'object') {
                setDatalist(inp, opt.datalist);
            }
            back.appendChild(a);
        }
    });
    document.body.appendChild(back);
}

function selectAndCopy(elm){
    window.getSelection().selectAllChildren(elm);
    document.execCommand('copy');
}

function post(url, data) {
    return new Promise((resolve, reject) => {
        sendAPI(url, data, 'POST')
        .then(res => resolve(res))
        .catch(err => reject(err));
    });
}

function get(url, object) {
    return new Promise((resolve, reject) => {
        if (object) {
            let query = new URLSearchParams(object).toString();
            sendAPI(url + '?' + query, null, 'GET')
            .then(res => resolve(res))
            .catch(err => reject(err));
        } else {
            sendAPI(url, null, 'GET')
            .then(res => resolve(res))
            .catch(err => reject(err));
        }
    });
}

function put(url, data) {
    return new Promise((resolve, reject) => {
        sendAPI(url, data, 'PUT')
        .then(res => resolve(res))
        .catch(err => reject(err));
    });
}

function del(url, data) {
    return new Promise((resolve, reject) => {
        sendAPI(url, data, 'DELETE')
        .then(res => resolve(res))
        .catch(err => reject(err));
    });
}

function sendAPI(url, data, method) {
    return new Promise((resolve, reject) => {
        let d = data;
        if (d == null && method != 'GET') d = new FormData();
        fetch(url, {
            method: method,
            body: d,
            credentials: 'include'
        }).then(res => {
            return res.text();
        }).then(txt => {
            try {
                resolve(JSON.parse(txt));
            } catch(err) {
                console.error(err);
                reject(err);
            }
        }).catch(err => {
            console.error(err);
            reject(err);
        });
    });
}

function postJson(url, obj) {
    return new Promise((resolve, reject) => {
        fetch(url, {
            'body': JSON.stringify(obj),
            'method': 'post',
            'credentials': 'include',
            'headers': {
                'Content-Type': 'application/json'
            }
        }).then(res => {
            return res.text();
        }).then(txt => {
            try {
                resolve(JSON.parse(txt));
            } catch(err) {
                console.error(err);
                reject(err);
            }
        }).catch(err => {
            console.error(err);
            reject(err);
        });
    });
}

function formDisabled(form, dis) {
	if (dis) {
		Array.from(form.getElementsByTagName('input')).forEach(elm => elm.setAttribute('disabled', ''));
		Array.from(form.getElementsByTagName('textarea')).forEach(elm => elm.setAttribute('disabled', ''));
		Array.from(form.getElementsByTagName('button')).forEach(elm => elm.setAttribute('disabled', ''));
		Array.from(form.getElementsByTagName('select')).forEach(elm => elm.setAttribute('disabled', ''));
        Array.from(form.querySelectorAll('input[type="checkbox"]')).forEach(elm => elm.setAttribute('onclick', 'return false;'));
        Array.from(form.querySelectorAll('input[type="radiobutton"]')).forEach(elm => elm.setAttribute('onclick', 'return false;'));
	} else {
		Array.from(form.getElementsByTagName('input')).forEach(elm => elm.removeAttribute('disabled'));
		Array.from(form.getElementsByTagName('textarea')).forEach(elm => elm.removeAttribute('disabled'));
		Array.from(form.getElementsByTagName('button')).forEach(elm => elm.removeAttribute('disabled'));
		Array.from(form.getElementsByTagName('select')).forEach(elm => elm.removeAttribute('disabled'));
        Array.from(form.querySelectorAll('input[type="checkbox"]')).forEach(elm => elm.removeAttribute('onclick'));
        Array.from(form.querySelectorAll('input[type="radiobutton"]')).forEach(elm => elm.removeAttribute('onclick'));
	}
}

function clearForm(form) {
    Array.from(form.getElementsByTagName('input')).forEach(elm => {
        if (elm.getAttribute('type') != 'button' && elm.getAttribute('type') != 'submit')
            elm.value = '';
    });
    Array.from(form.getElementsByTagName('textarea')).forEach(elm => elm.value = '');
    Array.from(form.getElementsByTagName('select')).forEach(elm => elm.selectedIndex = 0);
    Array.from(form.querySelectorAll('input[type="checkbox"]')).forEach(elm => elm.checked ? elm.click() : 0);
    Array.from(form.querySelectorAll('input[type="radiobutton"]')).forEach(elm => elm.removeAttribute('checked'));
}

function get2form(form) {
    let inputs = [];
    for (let i = 0; i < (inputs = form.getElementsByTagName('input')).length; i++) {
        if (inputs[i].getAttribute('type') == 'checkbox' || inputs[i].getAttribute('type') == 'radiobutton') {
            if (inputs[i].checked) inputs[i].click();
        }
    }
    new URL(location).searchParams.forEach((v, k) => {
        Array.from(form.querySelectorAll('[name="' + k + '"]')).forEach(elm => {
            if (elm.getAttribute('type') == 'checkbox' || elm.getAttribute('type') == 'radio') {
                if (elm.value == v) (!elm.checked ? elm.click() : 0);
            } else {
                elm.value = v;
            }
        });
    });
}

function get2object() {
    let ret = {};
    new URL(location).searchParams.forEach((v, k) => {
        ret[k] = v;
    });
    return ret;
}

function object2form(obj, form) {
    let inputs = [];
    for (let i = 0; i < (inputs = form.getElementsByTagName('input')).length; i++) {
        if (inputs[i].getAttribute('type') == 'checkbox' || inputs[i].getAttribute('type') == 'radiobutton') {
            inputs[i].checked = false;
        }
    }
    for (let i = 0; i < Object.keys(obj).length; i++) {
        let k = Object.keys(obj)[i];
        let v = obj[k];
        form.querySelectorAll('[name="' + k + '"]').forEach(elm => {
            if (elm.getAttribute('type') == 'checkbox' || elm.getAttribute('type') == 'radio') {
                if (elm.value == v) elm.checked = true;
            } else {
                elm.value = v;
            }
        });
    }
}

function form2object(form) {
    let obj = {};
    let data = new FormData(form);
    data.forEach((v, k) => {
        obj[k] = v;
    });
    return obj;
}

function getParentByTagName(elm, name) {
    for (let i = 0; elm.parentNode.tagName.toLowerCase() != name.toLowerCase(); i++) {
        elm = elm.parentNode;
        if (i > 100) {
            return null;
        }
    }
    return elm.parentNode;
}

setZero = i => (i < 10 ? '0' : '') + i;

function inputClear(formname_inputname) {
    let fi = formname_inputname.split('.');
    let inp = document.querySelector('form[name="' + fi[0] + '"] [name="' + fi[1] + '"]');
    inp.value = '';
    inp.dispatchEvent(new Event('change'));
}

function nohup(err) {
    let data = new FormData();
    data.append('error', err.message + '\n' + err.stack);
    post('/nohup', data);
}

/**
 * アルファベットの文字列をひらがなに変換する関数
 * 正規表現と文字コードの規則性を活用、例外処理も追加
 * @param {string} text - 変換したいアルファベット文字列
 * @return {string} - ひらがなに変換された文字列
 */
function alphabetToHiragana(text) {
    // 文字列を小文字に変換し、パターンマッチング用に準備
    const roma = text.toLowerCase();
    
    // パターン別の置換ルール（優先度順）
    const rules = [
      // 特殊な変換（優先度高）
      [/([kgszjtdnhfmbprlckgszjtdnhfmbprlc])\1/g, 'っ$1'],  // 促音: kk→っk
      [/nn|n(?![aiueoy])/g, 'ん'],                          // ん: nn, n+
      
      // 拗音
      [/([kszjtcnhfmbprlg])y([auo])/g, (_,c,v) => {
        // 行の決定（「き」などのベース文字を取得）
        const base = {
          k:'き',s:'し',t:'ち',j:'じ',c:'ち',
          n:'に',h:'ひ',f:'ふ',m:'み',
          y:'い',r:'り',g:'ぎ',z:'じ',
          d:'ぢ',b:'び',p:'ぴ'
        }[c];
        // 拗音の小文字部分
        const small = {'a':'ゃ','u':'ゅ','o':'ょ'}[v];
        return base + small;
      }],
      
      // 特殊なケース
      [/shi/g, 'し'], [/chi|ti/g, 'ち'], [/tsu|tu/g, 'つ'],
      [/fu|hu/g, 'ふ'], [/ji|zi/g, 'じ'],
      
      // 直接マッピング（文字コードの例外処理のため）
      [/ka/g, 'か'], [/ki/g, 'き'], [/ku/g, 'く'], [/ke/g, 'け'], [/ko/g, 'こ'],
      [/sa/g, 'さ'], [/si/g, 'し'], [/su/g, 'す'], [/se/g, 'せ'], [/so/g, 'そ'],
      [/ta/g, 'た'], [/ti/g, 'ち'], [/tu/g, 'つ'], [/te/g, 'て'], [/to/g, 'と'],
      [/na/g, 'な'], [/ni/g, 'に'], [/nu/g, 'ぬ'], [/ne/g, 'ね'], [/no/g, 'の'],
      [/ha/g, 'は'], [/hi/g, 'ひ'], [/hu/g, 'ふ'], [/he/g, 'へ'], [/ho/g, 'ほ'],
      [/ma/g, 'ま'], [/mi/g, 'み'], [/mu/g, 'む'], [/me/g, 'め'], [/mo/g, 'も'],
      [/ya/g, 'や'], [/yu/g, 'ゆ'], [/yo/g, 'よ'],
      [/ra/g, 'ら'], [/ri/g, 'り'], [/ru/g, 'る'], [/re/g, 'れ'], [/ro/g, 'ろ'],
      [/wa/g, 'わ'], [/wo/g, 'を'],
      [/ga/g, 'が'], [/gi/g, 'ぎ'], [/gu/g, 'ぐ'], [/ge/g, 'げ'], [/go/g, 'ご'],
      [/za/g, 'ざ'], [/zi/g, 'じ'], [/zu/g, 'ず'], [/ze/g, 'ぜ'], [/zo/g, 'ぞ'],
      [/da/g, 'だ'], [/di/g, 'ぢ'], [/du/g, 'づ'], [/de/g, 'で'], [/do/g, 'ど'],
      [/ba/g, 'ば'], [/bi/g, 'び'], [/bu/g, 'ぶ'], [/be/g, 'べ'], [/bo/g, 'ぼ'],
      [/pa/g, 'ぱ'], [/pi/g, 'ぴ'], [/pu/g, 'ぷ'], [/pe/g, 'ぺ'], [/po/g, 'ぽ'],
      
      // 母音のみ
      [/a/g, 'あ'], [/i/g, 'い'], [/u/g, 'う'], [/e/g, 'え'], [/o/g, 'お'],
      
      // 記号類
      [/-/g, 'ー'], [/\./g, '。'], [/,/g, '、']
    ];
    
    // ルールを順番に適用
    return rules.reduce((result, [pattern, replacement]) => 
      result.replace(pattern, replacement), roma);
}