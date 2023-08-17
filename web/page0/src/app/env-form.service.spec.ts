import { HttpClientTestingModule } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

import { EnvFormService } from './env-form.service';

describe('EnvFormService', () => {
  let service: EnvFormService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [],
    });

    service = TestBed.inject(EnvFormService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
