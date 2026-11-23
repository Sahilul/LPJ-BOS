<?php
/** Satu isian form. Variabel: name, label, value, type?, options?, required?, hint?, mono?, span2?, extra? */
$type = $type ?? 'text';
$value = $value ?? '';
$err = field_error($name);
$id = 'f-' . $name;
$attrs = !empty($required) ? ' required' : '';
$attrs .= $err ? ' aria-invalid="true" aria-describedby="' . $id . '-err"' : (!empty($hint) ? ' aria-describedby="' . $id . '-hint"' : '');
$attrs .= !empty($extra) ? ' ' . $extra : '';
$cls = 'input' . (!empty($mono) ? ' mono' : '');
?>
<div class="field<?= !empty($span2) ? ' span-2' : '' ?>">
  <?php if ($type === 'bool'): ?>
    <span class="label" aria-hidden="true">&nbsp;</span>
    <label class="check" for="<?= $id ?>">
      <input type="hidden" name="<?= e($name) ?>" value="0">
      <input type="checkbox" id="<?= $id ?>" name="<?= e($name) ?>" value="1"<?= $value ? ' checked' : '' ?><?= $attrs ?>>
      <span><?= e($label) ?></span>
    </label>
  <?php else: ?>
    <label for="<?= $id ?>"><?= e($label) ?><?= !empty($required) ? '<span class="req" aria-hidden="true">*</span>' : '' ?></label>
    <?php if ($type === 'textarea'): ?>
      <textarea class="<?= $cls ?>" id="<?= $id ?>" name="<?= e($name) ?>" rows="3"<?= $attrs ?>><?= e($value) ?></textarea>
    <?php elseif ($type === 'select'): ?>
      <select class="<?= $cls ?>" id="<?= $id ?>" name="<?= e($name) ?>"<?= $attrs ?>>
        <?php foreach ($options as $k => $v): ?>
          <option value="<?= e($k) ?>"<?= (string) $k === (string) $value ? ' selected' : '' ?>><?= e($v) ?></option>
        <?php endforeach; ?>
      </select>
    <?php elseif ($type === 'money'): ?>
      <?php $mv = is_numeric($value) ? (float) $value : (float) digits((string) $value); ?>
      <div class="input-group">
        <span class="addon">Rp</span>
        <input class="input mono" id="<?= $id ?>" name="<?= e($name) ?>" value="<?= $mv > 0 ? angka($mv) : '' ?>" inputmode="numeric" data-money autocomplete="off"<?= $attrs ?>>
      </div>
    <?php elseif ($type === 'percent'): ?>
      <div class="input-group">
        <input class="input mono" id="<?= $id ?>" name="<?= e($name) ?>" value="<?= e(str_replace('.', ',', (string) (float) $value)) ?>" inputmode="decimal" autocomplete="off"<?= $attrs ?>>
        <span class="addon end">%</span>
      </div>
    <?php else: ?>
      <input class="<?= $cls ?>" id="<?= $id ?>" name="<?= e($name) ?>" type="<?= e($type) ?>" value="<?= e($value) ?>"<?= $attrs ?>>
    <?php endif; ?>
  <?php endif; ?>
  <?php if ($err): ?>
    <span class="error" id="<?= $id ?>-err"><?= e($err) ?></span>
  <?php elseif (!empty($hint)): ?>
    <span class="hint" id="<?= $id ?>-hint"><?= e($hint) ?></span>
  <?php endif; ?>
</div>
