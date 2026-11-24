(() => {
  // Konfirmasi sebelum aksi berbahaya: <form data-confirm="...">
  document.addEventListener('submit', (ev) => {
    const msg = ev.target.dataset.confirm;
    if (msg && !window.confirm(msg)) ev.preventDefault();
  });

  // Tutup pesan flash
  document.addEventListener('click', (ev) => {
    const btn = ev.target.closest('[data-dismiss]');
    if (btn) btn.closest('.alert')?.remove();
  });
  setTimeout(() => document.querySelectorAll('.alert-success').forEach((a) => a.remove()), 6000);

  let activeSelectControl = null;
  let selectControlId = 0;
  const selectOptions = (control) => [...control.menu.querySelectorAll('[role="option"]:not([hidden]):not([aria-disabled="true"])')];

  const positionSelectMenu = (control) => {
    if (!control || control.menu.hidden) return;
    const rect = control.trigger.getBoundingClientRect();
    const below = window.innerHeight - rect.bottom - 8;
    const above = rect.top - 8;
    const placeAbove = below < 180 && above > below;
    const available = Math.max(120, Math.min(300, placeAbove ? above : below));
    const width = Math.min(rect.width, window.innerWidth - 16);
    control.menu.style.width = `${width}px`;
    control.menu.style.maxHeight = `${available}px`;
    control.menu.style.left = `${Math.max(8, Math.min(rect.left, window.innerWidth - width - 8))}px`;
    const height = Math.min(control.menu.scrollHeight, available);
    control.menu.style.top = `${placeAbove ? Math.max(8, rect.top - height - 6) : rect.bottom + 6}px`;
  };

  const closeSelectControl = (control, restoreFocus = false) => {
    if (!control) return;
    control.menu.hidden = true;
    control.wrapper.classList.remove('is-open');
    control.trigger.setAttribute('aria-expanded', 'false');
    if (activeSelectControl === control) activeSelectControl = null;
    if (restoreFocus) control.trigger.focus();
  };

  const updateSelectControl = (control) => {
    const { select, trigger, value, options, wrapper } = control;
    const selected = select.selectedOptions[0];
    value.textContent = selected?.textContent.trim() || 'Pilih...';
    const label = select.getAttribute('aria-label') || select.dataset.selectLabel || select.name;
    trigger.setAttribute('aria-label', `${label}: ${value.textContent}`);
    trigger.disabled = select.disabled;
    trigger.setAttribute('aria-invalid', select.getAttribute('aria-invalid') || 'false');
    control.list.setAttribute('aria-label', label);
    options.forEach((option, index) => {
      const source = select.options[index];
      option.setAttribute('aria-selected', String(source.selected));
      option.setAttribute('aria-disabled', String(source.disabled));
      option.hidden = source.hidden;
    });
    wrapper.classList.toggle('is-disabled', select.disabled);
  };

  const enhanceSelect = (select) => {
    if (select.dataset.selectEnhanced) return;
    const wasRequired = select.required;
    const associatedLabel = select.id
      ? document.querySelector(`label[for="${CSS.escape(select.id)}"]`)
      : null;
    select.dataset.selectLabel = associatedLabel?.textContent.trim() || select.getAttribute('aria-label') || select.name;
    const wrapper = document.createElement('div');
    wrapper.className = 'select-control';
    select.before(wrapper);
    wrapper.append(select);
    select.dataset.selectEnhanced = 'true';
    if (wasRequired) {
      select.dataset.selectRequired = 'true';
      select.required = false;
    }

    const id = `custom-select-${++selectControlId}`;
    const trigger = document.createElement('button');
    trigger.className = 'select-trigger';
    trigger.type = 'button';
    trigger.dataset.selectTrigger = '';
    trigger.id = select.id ? `${select.id}-custom` : `custom-select-${selectControlId + 1}-trigger`;
    if (associatedLabel) associatedLabel.htmlFor = trigger.id;
    trigger.setAttribute('aria-haspopup', 'listbox');
    trigger.setAttribute('aria-expanded', 'false');
    trigger.setAttribute('aria-controls', `${id}-listbox`);
    if (select.dataset.selectRequired) trigger.setAttribute('aria-required', 'true');
    if (select.getAttribute('aria-describedby')) {
      trigger.setAttribute('aria-describedby', select.getAttribute('aria-describedby'));
    }
    const value = document.createElement('span');
    value.className = 'select-value';
    trigger.append(value);
    const menu = document.createElement('div');
    menu.className = 'select-menu';
    menu.id = `${id}-menu`;
    menu.hidden = true;

    const searchOptions = select.options.length > 8;
    let search = null;
    if (searchOptions) {
      search = document.createElement('input');
      search.className = 'select-search';
      search.type = 'search';
      search.placeholder = 'Cari pilihan...';
      search.setAttribute('aria-label', 'Cari pilihan');
      search.dataset.selectSearch = '';
      menu.append(search);
    }

    const list = document.createElement('div');
    list.className = 'select-options';
    list.id = `${id}-listbox`;
    list.setAttribute('role', 'listbox');
    const options = [...select.options].map((source, index) => {
      const option = document.createElement('div');
      option.className = 'select-option';
      option.id = `${id}-option-${index}`;
      option.setAttribute('role', 'option');
      option.setAttribute('tabindex', '-1');
      option.dataset.value = source.value;
      option.textContent = source.textContent.trim();
      list.append(option);
      return option;
    });
    const empty = document.createElement('div');
    empty.className = 'select-empty';
    empty.textContent = 'Tidak ada pilihan yang cocok.';
    empty.hidden = true;
    list.append(empty);
    menu.append(list);
    wrapper.append(trigger);
    document.body.append(menu);

    const control = { select, wrapper, trigger, value, menu, list, search, options };
    select._selectControl = control;
    select.dataset.selectMenuId = menu.id;
    select.classList.add('select-native');
    select.tabIndex = -1;
    select.setAttribute('aria-hidden', 'true');

    const focusSelected = (direction = 1) => {
      const available = selectOptions(control);
      if (!available.length) return;
      const selected = available.findIndex((option) => option.getAttribute('aria-selected') === 'true');
      const option = available[selected >= 0 ? selected : direction > 0 ? 0 : available.length - 1];
      option.focus();
      option.scrollIntoView({ block: 'nearest' });
    };
    const open = (direction = 1) => {
      if (select.disabled) return;
      if (activeSelectControl && activeSelectControl !== control) closeSelectControl(activeSelectControl);
      activeSelectControl = control;
      menu.hidden = false;
      wrapper.classList.add('is-open');
      trigger.setAttribute('aria-expanded', 'true');
      if (search) {
        search.value = '';
        options.forEach((option) => { option.hidden = false; });
        empty.hidden = true;
      }
      positionSelectMenu(control);
      if (search) search.focus();
      else focusSelected(direction);
    };
    const choose = (option) => {
      if (!option || option.getAttribute('aria-disabled') === 'true') return;
      select.value = option.dataset.value;
      select.dispatchEvent(new Event('change', { bubbles: true }));
      closeSelectControl(control, true);
    };
    const moveOption = (current, direction) => {
      const available = selectOptions(control);
      if (!available.length) return;
      const index = available.indexOf(current);
      const next = index < 0
        ? (direction > 0 ? 0 : available.length - 1)
        : Math.max(0, Math.min(available.length - 1, index + direction));
      available[next].focus();
      available[next].scrollIntoView({ block: 'nearest' });
    };
    const filterOptions = () => {
      const query = search.value.trim().toLocaleLowerCase('id');
      let found = false;
      options.forEach((option) => {
        option.hidden = !option.textContent.toLocaleLowerCase('id').includes(query);
        if (!option.hidden) found = true;
      });
      empty.hidden = found;
      positionSelectMenu(control);
    };

    trigger.addEventListener('click', () => {
      if (menu.hidden) open();
      else closeSelectControl(control);
    });
    trigger.addEventListener('keydown', (event) => {
      if (['ArrowDown', 'ArrowUp', 'Enter', ' '].includes(event.key)) {
        event.preventDefault();
        open(event.key === 'ArrowUp' ? -1 : 1);
      }
    });
    menu.addEventListener('click', (event) => {
      const option = event.target.closest('[role="option"]');
      if (option) choose(option);
    });
    menu.addEventListener('keydown', (event) => {
      const option = event.target.closest('[role="option"]');
      if (event.key === 'Escape') {
        event.preventDefault();
        closeSelectControl(control, true);
      } else if (option && event.key === 'ArrowDown') {
        event.preventDefault();
        moveOption(option, 1);
      } else if (option && event.key === 'ArrowUp') {
        event.preventDefault();
        moveOption(option, -1);
      } else if (option && event.key === 'Home') {
        event.preventDefault();
        selectOptions(control)[0]?.focus();
      } else if (option && event.key === 'End') {
        event.preventDefault();
        selectOptions(control).at(-1)?.focus();
      } else if (option && ['Enter', ' '].includes(event.key)) {
        event.preventDefault();
        choose(option);
      } else if (event.key === 'Tab') {
        event.preventDefault();
        const focusable = [...document.querySelectorAll('a[href], button:not(:disabled), input:not(:disabled), select:not(:disabled), [tabindex="0"]')]
          .filter((element) => !element.classList.contains('select-native') && !element.closest('[hidden]') && element.offsetWidth);
        const index = focusable.indexOf(trigger);
        const next = index + (event.shiftKey ? -1 : 1);
        focusable[(next + focusable.length) % focusable.length]?.focus();
        closeSelectControl(control);
      }
    });
    search?.addEventListener('input', filterOptions);
    search?.addEventListener('keydown', (event) => {
      if (event.key === 'ArrowDown') {
        event.preventDefault();
        focusSelected();
      } else if (event.key === 'Escape') {
        event.preventDefault();
        closeSelectControl(control, true);
      } else if (event.key === 'Enter') {
        event.preventDefault();
        choose(selectOptions(control)[0]);
      } else if (event.key === 'Tab') {
        event.preventDefault();
        const focusable = [...document.querySelectorAll('a[href], button:not(:disabled), input:not(:disabled), select:not(:disabled), [tabindex="0"]')]
          .filter((element) => !element.classList.contains('select-native') && !element.closest('[hidden]') && element.offsetWidth);
        const index = focusable.indexOf(trigger);
        const next = index + (event.shiftKey ? -1 : 1);
        focusable[(next + focusable.length) % focusable.length]?.focus();
        closeSelectControl(control);
      }
    });
    select.addEventListener('change', () => {
      if (select.value) select.removeAttribute('aria-invalid');
      updateSelectControl(control);
      if (select.value) wrapper.querySelector('.select-validation')?.remove();
    });
    updateSelectControl(control);
  };

  const enhanceSelects = (root) => {
    if (root.matches?.('select')) enhanceSelect(root);
    root.querySelectorAll?.('select').forEach(enhanceSelect);
  };
  const setSelectValue = (select, value) => {
    if (!select) return;
    select.value = value;
    select.dispatchEvent(new Event('change', { bubbles: true }));
  };
  document.querySelectorAll('[data-bku-date]').forEach((input) => {
    input.addEventListener('change', () => input.form?.requestSubmit());
  });
  const focusFirstRowField = (row) => {
    const field = row.querySelector('input:not([type="hidden"]), select');
    if (field?.matches('select')) field._selectControl?.trigger.focus();
    else field?.focus();
  };
  enhanceSelects(document);
  document.addEventListener('pointerdown', (event) => {
    if (activeSelectControl
      && !activeSelectControl.wrapper.contains(event.target)
      && !activeSelectControl.menu.contains(event.target)) {
      closeSelectControl(activeSelectControl);
    }
  });
  document.addEventListener('submit', (event) => {
    const form = event.target;
    if (form.noValidate) return;
    const missing = [...form.querySelectorAll('select[data-select-required]')]
      .find((select) => !select.disabled && !select.value);
    if (!missing) return;
    event.preventDefault();
    missing.setAttribute('aria-invalid', 'true');
    const control = missing._selectControl;
    control.trigger.setAttribute('aria-invalid', 'true');
    if (!control.wrapper.querySelector('.select-validation')) {
      const error = document.createElement('span');
      error.className = 'select-validation';
      error.textContent = 'Pilih salah satu opsi.';
      control.wrapper.after(error);
    }
    control.trigger.focus();
  }, true);
  document.addEventListener('scroll', () => positionSelectMenu(activeSelectControl), true);
  window.addEventListener('resize', () => positionSelectMenu(activeSelectControl));
  new MutationObserver((records) => {
    records.forEach((record) => {
      if (record.type === 'attributes' && record.target._selectControl) {
        updateSelectControl(record.target._selectControl);
      }
      record.addedNodes.forEach((node) => {
        if (node.nodeType === Node.ELEMENT_NODE) enhanceSelects(node);
      });
      record.removedNodes.forEach((node) => {
        if (node.nodeType !== Node.ELEMENT_NODE) return;
        const removed = node.matches('select[data-select-menu-id]')
          ? [node]
          : [...node.querySelectorAll('select[data-select-menu-id]')];
        removed.forEach((select) => {
          if (select.isConnected) return;
          const control = select._selectControl;
          if (control) closeSelectControl(control);
          document.getElementById(select.dataset.selectMenuId)?.remove();
        });
      });
    });
  }).observe(document.body, { childList: true, subtree: true, attributes: true, attributeFilter: ['disabled', 'aria-invalid'] });

  // Menu samping di layar kecil
  const toggle = document.querySelector('[data-nav-toggle]');
  if (toggle) {
    const setOpen = (open) => {
      document.body.classList.toggle('nav-open', open);
      toggle.setAttribute('aria-expanded', String(open));
      toggle.setAttribute('aria-label', open ? 'Tutup menu' : 'Buka menu');
    };
    toggle.addEventListener('click', () => setOpen(!document.body.classList.contains('nav-open')));
    document.addEventListener('keydown', (ev) => ev.key === 'Escape' && setOpen(false));
    document.addEventListener('click', (ev) => {
      if (document.body.classList.contains('nav-open') && !ev.target.closest('.side, [data-nav-toggle]')) setOpen(false);
    });
  }

  // Pratinjau gambar sebelum diunggah: <input type=file data-preview="#id-img">
  document.querySelectorAll('input[type=file]').forEach((input) => {
    const drop = input.closest('.drop');
    const name = drop?.querySelector('.name');
    input.addEventListener('change', () => {
      const file = input.files[0];
      if (name) name.textContent = file ? file.name : name.dataset.empty || '';
      const target = input.dataset.preview && document.querySelector(input.dataset.preview);
      if (target && file && file.type.startsWith('image/')) {
        const url = URL.createObjectURL(file);
        target.innerHTML = '';
        const img = new Image();
        img.src = url;
        img.alt = 'Pratinjau';
        target.appendChild(img);
      }
      if (input.dataset.autosubmit !== undefined && file) input.form.requestSubmit();
    });
    if (drop) {
      ['dragenter', 'dragover'].forEach((t) => drop.addEventListener(t, () => drop.classList.add('is-over')));
      ['dragleave', 'drop'].forEach((t) => drop.addEventListener(t, () => drop.classList.remove('is-over')));
    }
  });

  // Format rupiah saat mengetik: <input data-money> (juga baris yang ditambah belakangan)
  const fmtMoney = (input) => {
    const d = input.value.replace(/\D/g, '').replace(/^0+(?=\d)/, '');
    input.value = d ? Number(d).toLocaleString('id-ID') : '';
  };
  document.addEventListener('input', (ev) => ev.target.matches('input[data-money]') && fmtMoney(ev.target));
  document.querySelectorAll('input[data-money]').forEach(fmtMoney);

  const uang = (v) => Number(String(v || '').replace(/\D/g, '')) || 0;
  const desimal = (v) => Number(String(v || '').replace(/[\s.]/g, '').replace(',', '.')) || 0;
  const rp = (n) => (n < 0 ? '−Rp' : 'Rp') + Math.abs(Math.round(n)).toLocaleString('id-ID');

  // Paket rencana: jumlah per periode × frekuensi
  document.querySelectorAll('[data-rencana-paket]').forEach((form) => {
    const jenis = form.querySelector('[data-jenis-target]');
    const rekening = form.querySelector('[data-rekening]');
    const frekuensi = form.querySelector('[data-frekuensi]');
    const totalOut = form.querySelector('[data-package-total]');
    let counter = 1000;

    const toggle = () => {
      form.querySelectorAll('[data-paket-only]').forEach((el) => {
        const show = el.dataset.paketOnly === 'barang' ? jenis.value === 'barang' : jenis.value !== 'barang';
        el.hidden = !show;
        el.querySelectorAll('input, select, textarea').forEach((input) => { input.disabled = !show; });
      });
    };
    const calc = () => {
      let total = 0;
      const activeRows = jenis.value === 'barang'
        ? form.querySelector('[data-package-rows=items]')
        : form.querySelector('[data-package-rows=penerima]');
      activeRows?.querySelectorAll('tbody tr').forEach((tr) => {
        const subtotal = Math.round(desimal(tr.querySelector('[data-vol]').value) * uang(tr.querySelector('[data-harga]').value));
        tr.querySelector('[data-sub]').textContent = rp(subtotal);
        total += subtotal;
      });
      totalOut.textContent = rp(total * Math.max(0, +frekuensi.value || 0));
    };
    const addRow = (editor) => {
      const html = editor.querySelector('template').innerHTML.replaceAll('__i__', String(counter++));
      const tbody = editor.querySelector('tbody');
      tbody.insertAdjacentHTML('beforeend', html);
      const row = tbody.lastElementChild;
      enhanceSelects(row);
      return row;
    };

    form.addEventListener('click', (ev) => {
      const add = ev.target.closest('[data-row-add]');
      if (add && add.closest('[data-package-rows]')) {
        focusFirstRowField(addRow(add.closest('[data-package-rows]')));
        calc();
        return;
      }
      const remove = ev.target.closest('[data-row-remove]');
      if (remove && remove.closest('[data-package-rows]')) {
        const editor = remove.closest('[data-package-rows]');
        const tbody = editor.querySelector('tbody');
        remove.closest('tr').remove();
        if (!tbody.children.length) addRow(editor);
        calc();
      }
    });
    form.addEventListener('input', calc);
    form.addEventListener('change', (ev) => {
      if (ev.target === jenis) toggle();
      if (ev.target === rekening) {
        const selectedJenis = rekening.selectedOptions[0]?.dataset.jenis;
        if (selectedJenis) setSelectValue(jenis, selectedJenis);
      }
      calc();
    });
    toggle();
    calc();
  });

  // Mutasi kas: uraian bawaan mengikuti jenis
  document.querySelectorAll('[data-mutasi]').forEach((form) => {
    const map = JSON.parse(form.dataset.mutasi || '{}');
    const jenis = form.querySelector('[name=jenis]');
    const uraian = form.querySelector('[name=uraian]');
    jenis?.addEventListener('change', () => {
      if (uraian && (!uraian.value || Object.values(map).includes(uraian.value))) uraian.value = map[jenis.value] || '';
    });
  });

  // Catat belanja
  const form = document.querySelector('[data-belanja]');
  if (form) {
    const $ = (sel, root = form) => root.querySelector(sel);
    const $$ = (sel, root = form) => [...root.querySelectorAll(sel)];
    const rencana = $('[data-rencana]');
    const metode = $('[data-metode]');
    // Selector harus spesifik: elemen <option> rencana juga memakai atribut
    // data-toko, sehingga '[data-toko]' bisa salah menunjuk option.
    const toko = $('select[data-toko]');
    const uraian = $('input[data-uraian]');
    const periode = $('[data-periode]');
    const periodeHint = $('[data-periode-hint]');
    let autoUraian = rencana.selectedOptions[0]?.dataset.uraian || '';
    const tarif = { ppn: +form.dataset.tarifPpn, pph22: +form.dataset.tarifPph22, pph23: +form.dataset.tarifPph23 };
    const batas = +form.dataset.batas;
    const batasPPh22 = +form.dataset.batasPph22;
    // persen: tampilkan tarif tanpa nol berlebih, mis. 1,5% atau 2%.
    const persen = (n) => `${String(Number(n.toFixed(2))).replace('.', ',')}%`;
    const danaBOS = () => $('[data-dana-bos]')?.checked ?? true;
    const viaSiplah = () => $('[data-siplah]')?.checked ?? false;
    // Nilai nota sudah termasuk PPN, jadi DPP = 11/111 bagian dari pembayaran
    // (PMK 59/2022 Lampiran VI.B.1).
    const dppDari = (bruto) => Math.round(bruto * 100 / (100 + tarif.ppn));
    // Tarif PPh 22 dan PPh 23 naik 100% bila rekanan tidak dapat menunjukkan NPWP.
    const tarifPph = (dasar, punyaNPWP) => (punyaNPWP ? dasar : dasar * 2);
    const pasal17 = (pkp) => {
      const lap = [[0, 60e6, 0.05], [60e6, 250e6, 0.15], [250e6, 500e6, 0.25], [500e6, 5e9, 0.3], [5e9, Infinity, 0.35]];
      let p = 0;
      for (const [bawah, atas, r] of lap) {
        if (pkp <= bawah) break;
        p += (Math.min(pkp, atas) - bawah) * r;
      }
      return Math.round(p);
    };
    // PPh 21 per rezim PMK 168/2023. Jenis penerima ditandai pada opsi penerima.
    const pph21 = (bruto, opt, hari) => {
      if (!bruto) return 0;
      const jenis = opt?.dataset?.jenisPen || 'tidak_tetap';
      const gol = opt?.dataset?.gol || '';
      if (jenis === 'pns') {
        if (/^(I|II)/.test(gol)) return 0;
        if (/^III/.test(gol)) return Math.round(bruto * 0.05);
        if (/^IV/.test(gol)) return Math.round(bruto * 0.15);
        return 0;
      }
      if (jenis === 'bukan_pegawai') return pasal17(bruto * 0.5);
      if (jenis === 'peserta') return pasal17(bruto);
      if (jenis === 'tetap') return pasal17(bruto);
      // Pegawai tidak tetap: tarif efektif harian bila upah sehari <= Rp2.500.000.
      const sehari = bruto / Math.max(1, hari || 1);
      if (sehari > 2500000) return pasal17(bruto * 0.5);
      return sehari <= 450000 ? 0 : Math.round(bruto * 0.005);
    };
    const out = (k) => $(`[data-out="${k}"]`);
    let counter = 1000;

    const jenisNow = () => rencana.selectedOptions[0]?.dataset.jenis || 'barang';

    const toggle = () => {
      const j = jenisNow();
      const show = { barang: j === 'barang', orang: j !== 'barang', honor: j === 'honor' };
      $$('[data-only]').forEach((el) => {
        const key = el.dataset.only;
        if (!(key in show)) return;
        el.hidden = !show[key];
        if (el.matches('section, .field')) $$('input, select, textarea', el).forEach((i) => { i.disabled = !show[key]; });
      });
    };

    const addRow = (editor) => {
      const tpl = editor.querySelector('template');
      const html = tpl.innerHTML.replaceAll('__i__', String(counter++));
      const tbody = editor.querySelector('tbody');
      tbody.insertAdjacentHTML('beforeend', html);
      const row = tbody.lastElementChild;
      enhanceSelects(row);
      if (editor.dataset.rows === 'pen') {
        const sat = rencana.selectedOptions[0]?.dataset.satuan;
        const s = row.querySelector('[data-satuan]');
        if (s && sat && !s.value) s.value = sat;
      }
      focusFirstRowField(row);
      return row;
    };

    const packageRows = (editor, rows, key) => {
      const tbody = editor.querySelector('tbody');
      tbody.innerHTML = '';
      rows.forEach((item) => {
        const row = addRow(editor);
        if (key === 'items') {
          row.querySelector('[name$="[nama]"]').value = item.nama || '';
          row.querySelector('[name$="[satuan]"]').value = item.satuan || '';
          const volume = row.querySelector('[data-vol]');
          volume.dataset.packageVolume = String(item.volume || 1);
          volume.value = String(item.volume || 1);
          row.querySelector('[data-harga]').value = item.harga > 0 ? item.harga.toLocaleString('id-ID') : '';
        } else {
          setSelectValue(row.querySelector('[data-pen]'), String(item.penerima_id || ''));
          row.querySelector('[name$="[satuan]"]').value = item.satuan || '';
          const volume = row.querySelector('[data-vol]');
          volume.dataset.packageVolume = String(item.volume || 1);
          volume.value = String(item.volume || 1);
          const price = row.querySelector('[data-harga]');
          price.value = item.tarif > 0 ? item.tarif.toLocaleString('id-ID') : '';
          if (!price.value) {
            const masterTarif = +row.querySelector('[data-pen]').selectedOptions[0]?.dataset.tarif || 0;
            if (masterTarif) price.value = masterTarif.toLocaleString('id-ID');
          }
        }
      });
      if (!rows.length) addRow(editor);
    };

    const applyPackage = (opt) => {
      if (!opt?.value) return;
      const packageData = JSON.parse(opt.dataset.paket || '{}');
      const count = Math.max(1, Math.min(12, +periode.value || 1));
      packageRows($('[data-rows=items]'), packageData.items || [], 'items');
      packageRows($('[data-rows=pen]'), packageData.penerima || [], 'penerima');
      if (toko) toko.value = opt.dataset.toko || '';
      scalePackageRows(count);
    };

    const scalePackageRows = (count) => {
      $$('[data-package-volume]').forEach((input) => {
        const base = desimal(input.dataset.packageVolume);
        input.value = String(base * count).replace('.', ',');
      });
    };

    const syncPeriods = () => {
      const opt = rencana.selectedOptions[0];
      if (!periode) return;
      const remaining = opt?.value ? Math.max(0, +opt.dataset.periodeSisa || 0) : 12;
      periode.max = String(Math.max(1, Math.min(12, remaining)));
      periode.disabled = remaining < 1;
      if (remaining > 0 && +periode.value > remaining) periode.value = String(remaining);
      if (remaining < 1) periode.value = '1';
      if (periodeHint) periodeHint.textContent = opt?.value
        ? (remaining > 0 ? `Maksimal ${remaining} bulan tersisa pada paket ini.` : 'Frekuensi paket sudah habis.')
        : 'Pilih jumlah bulan yang dibelanjakan dalam nota ini.';
    };

    const calc = () => {
      const j = jenisNow();
      let total = 0;
      let pajak = 0;
      if (j === 'barang') {
        $$('[data-rows=items] tbody tr').forEach((tr) => {
          const sub = Math.round(desimal(tr.querySelector('[data-vol]').value) * uang(tr.querySelector('[data-harga]').value));
          tr.querySelector('[data-sub]').textContent = rp(sub);
          total += sub;
        });
        const ppnOn = $('[data-pajak=ppn]').checked;
        const sel = toko?.options?.[toko.selectedIndex];
        const pkp = sel?.dataset?.pkp === '1';
        const punyaNPWP = sel?.dataset?.npwp === '1';
        const siplah = viaSiplah();
        const bos = danaBOS();
        // Batas Rp2.000.000 dibandingkan dengan nilai di luar PPN, bukan total nota
        // (PMK 59/2022 Pasal 18 ayat (1) huruf a).
        const dpp = ppnOn ? dppDari(total) : total;
        const tarifKini = {
          ppn: tarif.ppn,
          pph22: tarifPph(tarif.pph22, punyaNPWP),
          pph23: tarifPph(tarif.pph23, punyaNPWP),
        };
        // PPh 22 tidak dipungut untuk dana BOS maupun transaksi lewat SIPLah.
        const pph22Berlaku = !siplah && !bos && dpp > batasPPh22;
        $$('[data-pajak]').forEach((cb) => {
          const jenis = cb.dataset.pajak;
          let v = 0;
          if (cb.checked) {
            if (jenis === 'pph22' && !pph22Berlaku) v = 0;
            else v = Math.round(dpp * tarifKini[jenis] / 100);
          }
          const outEl = $(`[data-pajak-out="${jenis}"]`);
          if (outEl) outEl.textContent = rp(v);
          pajak += v;
        });
        const pph22Cb = $('[data-pajak=pph22]');
        if (pph22Cb) pph22Cb.closest('label').hidden = !pph22Berlaku;
        const pph22Label = $('[data-pph22-tarif]');
        if (pph22Label) pph22Label.textContent = persen(tarifKini.pph22);
        const pph23Label = $('[data-pph23-tarif]');
        if (pph23Label) pph23Label.textContent = persen(tarifKini.pph23);
        const saran = $('[data-pajak-saran]');
        if (saran) {
          const bisaPPN = pkp && !siplah && dpp > batas;
          const bagian = [];
          bagian.push(siplah
            ? 'Transaksi lewat SIPLah: PPN dan PPh 22 dipungut platform, bendahara tidak memungut lagi.'
            : (!pkp
              ? 'Rekanan bukan PKP: PPN tidak dipungut.'
              : (dpp > batas
                ? `Nilai di luar PPN di atas ${rp(batas)}: PPN dipungut (11/111 dari nota).`
                : `Nilai di luar PPN paling banyak ${rp(batas)}: PPN tidak dipungut.`)));
          if (!siplah) {
            bagian.push(bos
              ? 'Dana BOS: pembelian barang tidak dipungut PPh 22.'
              : (dpp > batasPPh22 ? 'PPh 22 dipungut 1,5% dari nilai di luar PPN.' : `PPh 22 tidak dipungut: nilai di luar PPN paling banyak ${rp(batasPPh22)}.`));
          }
          saran.textContent = bagian.join(' ');
          saran.classList.toggle('is-warn', bisaPPN && !$('[data-pajak=ppn]').checked);
        }
      } else {
        $$('[data-rows=pen] tbody tr').forEach((tr) => {
          const opt = tr.querySelector('[data-pen]').selectedOptions[0];
          const hariEl = tr.querySelector('[data-hari]');
          const hari = hariEl ? desimal(hariEl.value) : 0;
          const bruto = Math.round(desimal(tr.querySelector('[data-vol]').value) * uang(tr.querySelector('[data-harga]').value));
          const pph = j === 'honor' && opt?.value ? pph21(bruto, opt, hari) : 0;
          tr.querySelector('[data-sub]').textContent = rp(bruto);
          tr.querySelector('td[data-pph]').textContent = rp(pph);
          tr.querySelector('[data-netto]').textContent = rp(bruto - pph);
          total += bruto;
          pajak += pph;
        });
      }

      const opt = rencana.selectedOptions[0];
      const sisa = opt?.value ? +opt.dataset.sisa : null;
      const pagu = opt?.value ? +opt.dataset.pagu : 0;
      const m = metode.value;
      const saldo = +form.dataset[m === 'bank' ? 'saldoBank' : 'saldoTunai'];
      out('total').textContent = rp(total);
      out('sisa').textContent = sisa === null ? '—' : rp(sisa);
      out('after').textContent = sisa === null ? '—' : rp(sisa - total);
      out('pajak').textContent = rp(pajak);
      $('[data-only=pajak]').hidden = pajak <= 0;
      out('metode').textContent = m === 'bank' ? 'bank' : 'kas tunai';
      out('saldo').textContent = rp(saldo);
      const used = pagu > 0 ? Math.min(100, ((pagu - (sisa ?? 0) + total) / pagu) * 100) : 0;
      const bar = out('bar');
      bar.style.width = used + '%';

      let msg = '';
      if (sisa !== null && total > sisa + 0.001) msg = `Melebihi sisa pagu ${rp(sisa)}. Kurangi rincian atau pilih rencana lain.`;
      const bad = !!msg;
      if (!msg && total > saldo + 0.001) msg = `Saldo ${m === 'tunai' ? 'tunai' : 'bank'} saat ini ${rp(saldo)}, jadi BKU akan minus. Tetap bisa disimpan; tanggal lunas di kuitansi mengikuti tanggal dana cair.`;
      const box = out('msg');
      box.textContent = msg;
      box.hidden = !msg;
      box.classList.toggle('is-bad', bad);
      bar.parentElement.classList.toggle('is-bad', sisa !== null && total > sisa + 0.001);
      bar.parentElement.classList.toggle('is-full', sisa !== null && Math.abs(total - sisa) < 0.001 && total > 0);
    };

    rencana.addEventListener('change', () => {
      const opt = rencana.selectedOptions[0];
      if (opt?.value && uraian && (!uraian.value || uraian.value === autoUraian)) {
        uraian.value = opt.dataset.uraian || '';
        autoUraian = uraian.value;
      }
      if (opt?.dataset.satuan) $$('[data-rows=pen] [data-satuan]').forEach((s) => { if (!s.value) s.value = opt.dataset.satuan; });
      syncPeriods();
      if (!form.dataset.edit) applyPackage(opt);
      toggle();
      calc();
    });
    periode?.addEventListener('input', () => {
      scalePackageRows(Math.max(1, +periode.value || 1));
      calc();
    });

    form.addEventListener('click', (ev) => {
      const add = ev.target.closest('[data-row-add]');
      if (add) { addRow(add.closest('[data-rows]')); calc(); return; }
      const rm = ev.target.closest('[data-row-remove]');
      if (rm) {
        const tbody = rm.closest('tbody');
        const editor = rm.closest('[data-rows]');
        rm.closest('tr').remove();
        if (!tbody.children.length) addRow(editor);
        calc();
      }
    });

    form.addEventListener('change', (ev) => {
      const pen = ev.target.closest('[data-pen]');
      if (pen) {
        const tr = pen.closest('tr');
        const harga = tr.querySelector('[data-harga]');
        const t = +pen.selectedOptions[0]?.dataset.tarif || 0;
        if (t && !uang(harga.value)) harga.value = t.toLocaleString('id-ID');
      }
      calc();
    });
    form.addEventListener('input', calc);

    syncPeriods();
    toggle();
    calc();
  }

  // Filter otomatis saat pilihan berubah
  document.querySelectorAll('select[data-autosubmit]').forEach((s) => s.addEventListener('change', () => s.form.submit()));

  document.querySelectorAll('[data-rencana-pendapatan]').forEach((select) => {
    const form = select.form;
    const tahap = form.querySelector('[name="tahap"]');
    const uraian = form.querySelector('[name="uraian"]');
    const jumlah = form.querySelector('[name="jumlah"]');
    let autoUraian = select.selectedOptions[0]?.dataset.uraian || '';
    let autoJumlah = +select.selectedOptions[0]?.dataset.target || 0;

    select.addEventListener('change', () => {
      const option = select.selectedOptions[0];
      if (!option?.value) {
        return;
      }
      if (tahap) setSelectValue(tahap, option.dataset.tahap || tahap.value);
      if (uraian && (!uraian.value || uraian.value === autoUraian)) {
        uraian.value = option.dataset.uraian || uraian.value;
        autoUraian = uraian.value;
      }
      const currentJumlah = uang(jumlah?.value);
      if (jumlah && (!currentJumlah || currentJumlah === autoJumlah)) {
        autoJumlah = +option.dataset.target || 0;
        jumlah.value = autoJumlah ? autoJumlah.toLocaleString('id-ID') : '';
      } else {
        autoJumlah = +option.dataset.target || 0;
      }
    });
  });
})();
